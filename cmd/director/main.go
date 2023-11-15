/*
Copyright (c) Richard Kojedzinszky <richard@kojedz.in>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:
 1. Redistributions of source code must retain the above copyright
    notice, this list of conditions and the following disclaimer.
 2. Redistributions in binary form must reproduce the above copyright
    notice, this list of conditions and the following disclaimer in the
    documentation and/or other materials provided with the distribution.
 3. Neither the name of the University nor the names of its contributors
    may be used to endorse or promote products derived from this software
    without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE REGENTS AND CONTRIBUTORS “AS IS” AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
ARE DISCLAIMED.  IN NO EVENT SHALL THE REGENTS OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
SUCH DAMAGE.
*/

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/namsral/flag"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"go-dovecot-director/pkg/allocator/postgres"
	"go-dovecot-director/pkg/director"
	kpool "go-dovecot-director/pkg/pool/kubernetes"
)

var (
	kubeconfig = func() *string {
		if home := homedir.HomeDir(); home != "" {
			return flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
		}
		return flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}()

	namespace = flag.String("namespace", "", "Namespace of services to watch")
	service   = flag.String("service", "", "Service for backend PODs")

	directorListenAddress = flag.String("director-listen-address", ":8080", "Listen address for director requests")

	databaseHost     = flag.String("database-host", "postgres", "Postfixadmin database hostname")
	databasePort     = flag.Int("database-port", 5432, "Postfixadmin database port")
	databaseName     = flag.String("database-name", "postfixadmin", "Postfixadmin database name")
	databaseUser     = flag.String("database-user", "postfixadmin", "Postfixadmin database username")
	databasePassword = flag.String("database-password", "postfixadmin", "Postfixadmin database password")
)

func newClientSet() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	}

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func main() {
	flag.Parse()

	directorListener, err := net.Listen("tcp", *directorListenAddress)
	if err != nil {
		log.Fatal(err)
	}

	db, err := pgxpool.New(context.TODO(),
		fmt.Sprintf(
			"host=%s port=%d database=%s user=%s password=%s sslmode=disable pool_max_conns=2",
			*databaseHost, *databasePort, *databaseName, *databaseUser, *databasePassword,
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	client, err := newClientSet()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := kpool.New(client, *namespace, *service)
	if err != nil {
		log.Fatal(err)
	}

	allocator := postgres.New(db, pool)
	dir := director.New(allocator)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}

	// start poolmonitor
	wg.Add(1)
	go func() {
		defer wg.Done()

		pool.Run(ctx)
	}()

	// start dovecot server
	wg.Add(1)
	go func() {
		defer wg.Done()

		dir.Serve(ctx, directorListener)
	}()

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGINT)
	<-sigchan

	log.Print("Exiting gracefully...")
	time.Sleep(5 * time.Second)

	cancel()

	wg.Wait()
}
