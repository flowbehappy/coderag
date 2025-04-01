# Copyright 2023 PingCAP, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import os
import mysql.connector
from contextlib import contextmanager


class DB:
    def __init__(self):
        self.tidb_host = os.getenv("TIDB_HOST", "127.0.0.1")
        self.tidb_port = int(os.getenv("TIDB_PORT", "4000"))
        self.tidb_user = os.getenv("TIDB_USER", "root")
        self.tidb_password = os.getenv("TIDB_PASSWORD", "")
        self.tidb_db_name = os.getenv("TIDB_DB_NAME", "test")
        self.ca_path = os.getenv("CA_PATH", "")
        with self.open_db() as (conn, cur):
            cur.execute(f'''CREATE TABLE IF NOT EXISTS {self.tidb_db_name}(
                message_id varchar (255),
                thread_id varchar (255),
                content varchar(255),
                create_time timestamp NOT NULL,
                update_time timestamp,
                PRIMARY KEY (message_id, thread_id));'''
                        )

    @contextmanager
    def open_db(self, autocommit: bool = True):
        db_conf = {
            "host": self.tidb_host,
            "port": self.tidb_port,
            "user": self.tidb_user,
            "password": self.tidb_password,
            "database": self.tidb_db_name,
            "autocommit": autocommit,
            # mysql-connector-python will use C extension by default,
            # to make this example work on all platforms more easily,
            # we choose to use pure python implementation.
            "use_pure": True,
        }
        if self.ca_path:
            db_conf["ssl_verify_cert"] = True
            db_conf["ssl_verify_identity"] = True
            db_conf["ssl_ca"] = self.ca_path
        with mysql.connector.connect(**db_conf) as conn:
            with conn.cursor(dictionary=True) as cur:
                yield conn, cur

    def insert(self, message_id, thread_id, content, create_time):
        with self.open_db() as (conn, cur):
            cur.execute(
                f"INSERT INTO {self.tidb_db_name} VALUES(%s, %s, %s, %s)", (message_id, thread_id, content, create_time))

    def update(self, message_id, thread_id, content, update_time):
        with self.open_db() as (conn, cur):
            cur.execute(
                f"UPDATE {self.tidb_db_name} SET content = %s, update_time = %s WHERE message_id = %s and thread_id = %s", (content, update_time, message_id, thread_id))

    def query(self, message_id, thread_id):
        with self.open_db() as (conn, cur):
            cur.execute(
                f"SELECT * from {self.tidb_db_name} WHERE message_id = %s and thread_id = %s", (message_id, thread_id))
            return cur.fetchone()
