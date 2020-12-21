/*
Copyright 2020 Dynatrace LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"net/http"

	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/namespace"
	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/nodes"
	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/oneagent"
	"github.com/Dynatrace/dynatrace-oneagent-operator/controllers/oneagentapm"
	"github.com/prometheus/common/log"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func startOperator(ns string, cfg *rest.Config) (manager.Manager, error) {
	log.Info(ns)
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Namespace:               ns,
		Scheme:                  scheme,
		MetricsBindAddress:      ":8080",
		Port:                    8383,
		LeaderElection:          true,
		LeaderElectionID:        "dynatrace-oneagent-operator-lock",
		LeaderElectionNamespace: ns,
	})
	if err != nil {
		return nil, err
	}

	log.Info("Registering Components.")

	for _, f := range []func(manager.Manager, string) error{
		oneagent.Add,
		oneagentapm.Add,
		namespace.Add,
		nodes.Add,
	} {
		if err := f(mgr, ns); err != nil {
			return nil, err
		}
	}

	go func() {
		log.Info("serving operator probe endpoint on :10080/healthz")
		err := http.ListenAndServe(":10080", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path == "/healthz" {
				writer.WriteHeader(http.StatusOK)
			} else {
				writer.WriteHeader(http.StatusNotFound)
			}
		}))
		if err != nil {
			log.Error(err, "encountered error while serving operator's probe endpoint")
		}
	}()

	return mgr, nil
}
