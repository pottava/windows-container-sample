import os
from datetime import datetime
from google.cloud import bigquery

def insert(rows):
    client = bigquery.Client()
    ds = client.dataset(os.environ['DATASET'])
    tbl = ds.table(os.environ['TABLE'])
    table = client.get_table(tbl)
    client.insert_rows(table, rows)

def main():
    name = os.environ['JOB_NAME']
    index = int(os.environ['VK_TASK_INDEX'])
    rows = [(name, index, u'started', datetime.utcnow())]
    insert(rows)

if __name__ == "__main__":
    main()
