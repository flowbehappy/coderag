#!/bin/bash
tiup playground nightly &

sleep 5
echo "Verifying TiDB is started..."
TIDB_HOST=127.0.0.1
TIDB_PORT=4000
i=0
while ! mysql -uroot -h${TIDB_HOST} -P${TIDB_PORT} --default-character-set utf8mb4 -e 'select * from mysql.tidb;'; do
	i=$((i + 1))
	if [ "$i" -gt 60 ]; then
		echo 'Failed to start upstream TiDB'
		exit 2
	fi
	sleep 2
done

 python feishu.py