for filling env file ==>
-driver: [mysql | postgres | sqlite]
-conn:   the connection string used by the driver.
          default for mysql:    root:@tcp(127.0.0.1:3306)/test
          default for postgres: postgres://postgres:postgres@127.0.0.1:5432/postgres