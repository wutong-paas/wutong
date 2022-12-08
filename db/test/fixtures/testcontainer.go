package fixtures

import (
	"context"
	"fmt"
	"time"

	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/wutong-paas/wutong/db"
	dbconfig "github.com/wutong-paas/wutong/db/config"
)

// InitDBManager initiates a db manager with a real mysql provided by testcontainers-go.
func InitDBManager() error {
	dbname := "region"
	rootpw := "wutong"

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "mariadb",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": rootpw,
			"MYSQL_DATABASE":      dbname,
		},
		Cmd: []string{"character-set-server=utf8mb4", "collation-server=utf8mb4_unicode_ci"},
	}
	mariadb, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return err
	}
	defer mariadb.Terminate(ctx)

	host, err := mariadb.Host(ctx)
	if err != nil {
		return err
	}
	port, err := mariadb.MappedPort(ctx, "3306")
	if err != nil {
		return err
	}

	connInfo := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", "root",
		rootpw, host, port.Int(), dbname)
	tryTimes := 3
	for {
		if err := db.CreateManager(dbconfig.Config{
			DBType:              "mysql",
			MysqlConnectionInfo: connInfo,
		}); err != nil {
			tryTimes = tryTimes - 1
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}
	return nil
}
