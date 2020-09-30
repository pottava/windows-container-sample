# サンプルアプリケーション

## 1. クラスタのセットアップ

### 1.1. gcloud のインストール

https://cloud.google.com/sdk/docs/quickstart-windows?hl=ja  
をご参照いただき、gcloud のインストールをお願いします。以後、PowerShell での実行を想定します。

```bash
$project_id = ""
$compute_region = "asia-northeast1"
$compute_zone = "asia-northeast1-c"
gcloud config set project ${project_id}
gcloud config set compute/region ${compute_region}
gcloud config set compute/zone ${compute_zone}
```

### 1.2. GKE クラスタの作成

基本的な設定値を指定し、

```bash
$cluster_name = "batch"
$cluster_ver = "1.16.13-gke.401"
$machine_type = "n1-standard-2"
```

デフォルトのノードプールを GCP の Container-Optimized OS (Linux) として GKE クラスタを作成します。

```bash
gcloud container clusters create ${cluster_name} --zone ${compute_zone} --cluster-version ${cluster_ver} --machine-type ${machine_type} --enable-ip-alias --preemptible --enable-autoscaling --num-nodes 1 --min-nodes 1 --max-nodes 3 --enable-autorepair --max-surge-upgrade 1 --max-unavailable-upgrade 0 --node-labels "os=cos" --enable-stackdriver-kubernetes --no-enable-autoupgrade --maintenance-window-start "2000-01-01T09:00:00-04:00" --maintenance-window-end "2000-01-01T17:00:00-04:00" --maintenance-window-recurrence 'FREQ=WEEKLY;BYDAY=SA,SU' --scopes "service-control,service-management,compute-rw,storage-ro,cloud-platform,logging-write,monitoring-write" --no-enable-basic-auth --no-issue-client-certificate
```

### 1.3. Volcano のインストール

kubectl での操作ができるようクラウドから kubeconfig を取得します。

```bash
gcloud container clusters get-credentials ${cluster_name}
```

以下のコマンドで Volcano をインストールしてください。

```bash
$volcano_version = "v1.0.1"
kubectl apply -f "https://raw.githubusercontent.com/volcano-sh/volcano/${volcano_version}/installer/volcano-development.yaml"
kubectl -n volcano-system get all
```

### 1.4. Windows ノードプールの追加

Windows ノードプールを追加します。  
（また、Windows の場合、GPU やプリエンプティブル VM、Workload Identity が利用できないことにご注意ください）  
Windows のバージョンマッピング（コンテナとして実行できるベースイメージと関係してきます）については以下をご参照ください。  
https://cloud.google.com/kubernetes-engine/docs/how-to/creating-a-cluster-windows?hl=ja#version_mapping

```bash
gcloud container node-pools create "${cluster_name}-win-ltsc" --cluster ${cluster_name} --machine-type ${machine_type} --image-type "WINDOWS_LTSC" --enable-autoscaling --num-nodes 1 --min-nodes 1 --max-nodes 10 --enable-autorepair --max-surge-upgrade 1 --max-unavailable-upgrade 0 --no-enable-autoupgrade --node-labels "os=win-ltsc" --metadata "disable-legacy-endpoints=true"
```

### 1.5. その他スケジュール上有用な設定・リソースを配置

プライオリティを設定します。

```bash
kubectl apply -f sample/00-common/priority.yaml
```

ノードの状態を確認します。

```bash
kubectl get nodes -o wide
```

## 2. クラスタのセットアップ

### 2.1. アプリケーションの build

```bash
cd sample/00-common
docker run --rm -it -v "<カレントディレクトリ>":C:\tmp -w C:\tmp golang:1.14.4-nanoserver-1809 cmd.exe
$ go build
$ exit
docker build -t "gcr.io/${project_id}/sample-apps:envs" .
```

### 2.2. アプリケーションの ship と deploy

sample/00-common/samples.yaml の @your-project-id を ${project_id} に置き換え、以下を実行します。

```bash
gcloud auth configure-docker
docker push "gcr.io/${project_id}/sample-apps:envs"
kubectl apply -f sample/00-common/samples.yaml
```

### 2.3. 結果の確認とお掃除

```bash
open "https://console.cloud.google.com/kubernetes/workload"
kubectl describe job.batch.volcano.sh sample
kubectl delete job.batch.volcano.sh sample
```

## 3. 並列分散処理の実行

### 3.1. Cloud Storage (GCS) へのファイルアップロード

```bash
$samle_bucket_name = "sample-<今日の日時>"
$samle_user_id = "user-0001"
gsutil mb -c STANDARD -l ${compute_region} gs://${samle_bucket_name}/
gsutil cp sample/01-task/input.csv gs://${samle_bucket_name}/${samle_user_id}/
gsutil cp sample/01-task/parameters.csv gs://${samle_bucket_name}/${samle_user_id}/
```

### 3.2. GKE から GCS へアクセスするための設定

GCP へのアクセス権限をもつサービスアカウントと、そのアクセスキーを作ります。

```bash
gcloud iam service-accounts create batch-sample
gcloud projects add-iam-policy-binding ${project_id} --member "serviceAccount:batch-sample@${project_id}.iam.gserviceaccount.com" --role roles/storage.admin
gcloud iam service-accounts keys create key.json --iam-account "batch-sample@${project_id}.iam.gserviceaccount.com"
```

それを GKE 上の Secret に格納します。

```bash
kubectl create secret generic batch-sample --from-file=key.json=key.json
kubectl describe secrets batch-sample
```

### 3.3. アプリケーションの build, ship & deploy

sample/01-task/task.yaml を以下の値に置き換え、後続のコマンドを実行します。

- @your-project-id > ${project_id}
- @your-bucket > ${samle_bucket_name}
- @your-user > ${samle_user_id}

```bash
cd sample/01-task
docker run --rm -it -v "<カレントディレクトリ>":C:\go\src\github.com\pottava\windows-container-sample -w C:\go\src\github.com\pottava\windows-container-sample golang:1.14.4-nanoserver-1809 cmd.exe
$ go mod vendor
$ go build
$ exit
docker build -t "gcr.io/${project_id}/sample-apps:01" .
docker push "gcr.io/${project_id}/sample-apps:01"
kubectl apply -f sample/01-task/task.yaml
```

### 3.3. 結果の確認とお掃除

```bash
open "https://console.cloud.google.com/kubernetes/workload"
kubectl describe job.batch.volcano.sh task01
kubectl delete job.batch.volcano.sh task01
```

## 4. サンプルのカスタマイズ

sample/02-task/task.yaml の値を置き換えます。

- @your-project-id > ${project_id}
- @your-bucket > ${samle_bucket_name}
- @your-user > ${samle_user_id}

### 4.1. ジョブ定義の基本

- `metadata.name`: ジョブ名として識別されます (ex. [user-id]-[project-id]-[start-date]-[seq], 0001-a-200615-01)
- `spec.plugins.env`: 必須。コンテナ側に `VK_TASK_INDEX` という環境変数が渡り、何個目のタスクかを判定できます
- `spec.affinity`: 計算クラスターの特定ノードで計算されるよう調整できます

```bash
kubectl apply -f sample/02-task/task.yaml
kubectl describe job.batch.volcano.sh 0001-a-200615-01
kubectl delete job.batch.volcano.sh 0001-a-200615-01
```

### 4.2. 並列数の調整、ジョブ投入

- `spec.tasks[].replicas`: 並列処理したいタスク数 (ex. 10000)
- `spec.minAvailable`: 処理開始並列数 (ex. 100 -> 100 並列流せるリソースがあれば処理開始、9900 は待機)
- `spec.maxRetry`: タスク失敗時のリトライ数上限

```bash
kubectl apply -f sample/02-task/task.yaml
```

### 4.3. 自動スケール

待機ジョブが増えてくるとクラスタのノードがスケールします。Windows ノードプールの `min/max-nodes` で事前に調整してください。
