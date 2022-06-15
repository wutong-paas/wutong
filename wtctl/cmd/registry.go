// Copyright (C) 2014-2021 Wutong Co., Ltd.
// WUTONG, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/config"
	"github.com/wutong-paas/wutong/wtctl/clients"
	"github.com/wutong-paas/wutong/wtctl/registry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NewCmdRegistry registry cmd
func NewCmdRegistry() cli.Command {
	c := cli.Command{
		Name:  "registry",
		Usage: "wtctl registry [command]",
		Subcommands: []cli.Command{
			{
				Name: "cleanup",
				Usage: `Clean up free images in the registry.
	The command 'wtctl registry cleanup' will delete the index of free images in registry.
	Then you have to exec the command below to remove blobs from the filesystem:
		bin/registry garbage-collect [--dry-run] /path/to/config.yml
	More Detail: https://docs.docker.com/registry/garbage-collection/#run-garbage-collection.
				`,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "namespace, ns",
						Usage:  "wutong namespace",
						EnvVar: "WTNamespace",
						Value:  "wt-system",
					},
				},
				Action: func(c *cli.Context) error {
					Common(c)

					namespace := c.String("namespace")
					var cluster wutongv1alpha1.WutongCluster
					if err := clients.WutongKubeClient.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: "wutongcluster"}, &cluster); err != nil {
						return errors.Wrap(err, "get configuration from wutong cluster")
					}

					dsn, err := databaseDSN(&cluster)
					if err != nil {
						return errors.Wrap(err, "get database dsn")
					}

					dbCfg := config.Config{
						MysqlConnectionInfo: dsn,
						DBType:              "mysql",
					}
					if err := db.CreateManager(dbCfg); err != nil {
						return errors.Wrap(err, "create database manager")
					}

					registryConfig := cluster.Spec.ImageHub
					cleaner, err := registry.NewRegistryCleaner(registryConfig.Domain, registryConfig.Username, registryConfig.Password)
					if err != nil {
						return errors.WithMessage(err, "create registry cleaner")
					}

					cleaner.Cleanup()

					return nil
				},
			},
		},
	}
	return c
}

func databaseDSN(wutongcluster *wutongv1alpha1.WutongCluster) (string, error) {
	database := wutongcluster.Spec.RegionDatabase
	if database != nil {
		return fmt.Sprintf("%s:%s@tcp(%s)/%s", database.Username, database.Password, database.Host, database.Name), nil
	}
	// default name of wt-db pod is wt-db-0
	pod, err := clients.K8SClient.CoreV1().Pods(wutongcluster.Namespace).Get(context.Background(), "wt-db-0", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "get pod wt-db-0")
	}
	host := pod.Status.PodIP
	name := "region"
	for _, ct := range pod.Spec.Containers {
		if ct.Name != "wt-db" {
			continue
		}
		for _, env := range ct.Env {
			if env.Name == "MYSQL_DATABASE" {
				name = env.Value
			}
		}
	}

	secret, err := clients.K8SClient.CoreV1().Secrets(wutongcluster.Namespace).Get(context.Background(), "wt-db", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "get secret wt-db")
	}
	username := string(secret.Data["mysql-user"])
	password := string(secret.Data["mysql-password"])

	return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s", username, password, host, name), nil
}
