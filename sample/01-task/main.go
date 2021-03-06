package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"cloud.google.com/go/storage"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
			log.Fatalln(err)
		}
	}()
	// 実行時引数を取得
	cfg := getConfig()
	ctx := getContext()
	client, bucket := getBucket(ctx, cfg)
	defer client.Close()

	// パラメタファイルの取得
	params := getParams(ctx, cfg, bucket)
	if len(params) == 0 {
		log.Fatalln("Invalid arguments")
	}
	// メインの演算
	result := calculate(ctx, cfg, bucket, params)
	fmt.Println(result)
}

type config struct {
	index   int
	bucket  string
	input   string
	params  string
	timeout time.Duration
}

func getConfig() config {
	idx, err := strconv.Atoi(os.Getenv("VK_TASK_INDEX"))
	if err != nil {
		log.Fatalln(err)
	}
	return config{
		index:  idx%3 + 1,
		bucket: os.Getenv("INPUT_BUCKET"),
		input:  os.Getenv("INPUT_FILE"),
		params: os.Getenv("PARAMETER_FILE"),
	}
}

func getContext() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5*time.Second))
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		os.Exit(1)
	}()
	return ctx
}

func getBucket(ctx context.Context, cfg config) (*storage.Client, *storage.BucketHandle) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	return client, client.Bucket(cfg.bucket)
}

func getParams(ctx context.Context, cfg config, bucket *storage.BucketHandle) []string {
	object := bucket.Object(cfg.params)
	raw, rErr := object.NewReader(ctx)
	if rErr != nil {
		log.Fatalln(rErr)
	}
	defer raw.Close()

	// タスクに割り振られた index 番目のパラメタを返す
	reader := csv.NewReader(raw)
	count := -1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			return []string{}
		}
		if err != nil {
			log.Fatal(err)
		}
		count++
		if count == cfg.index {
			return record
		}
	}
}

func calculate(ctx context.Context, cfg config, bucket *storage.BucketHandle, params []string) []string {
	object := bucket.Object(cfg.input)
	raw, rErr := object.NewReader(ctx)
	if rErr != nil {
		log.Fatalln(rErr)
	}
	defer raw.Close()

	// input ファイルの読み込み
	reader := csv.NewReader(raw)
	multiply := 1
	count := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if count > 0 && len(record) > 0 {
			candidate, err := strconv.Atoi(record[0])
			if err != nil {
				log.Fatalln(err)
			}
			multiply = candidate
		}
		count++
	}

	// 実際の計算を実行
	result := []string{}
	for _, param := range params {
		if value, err := strconv.Atoi(param); err == nil {
			result = append(result, fmt.Sprintf("%d", value*multiply))
		}
	}
	return result
}
