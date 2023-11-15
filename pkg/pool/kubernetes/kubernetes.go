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

package kubernetes

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"go-dovecot-director/pkg/pool"
)

var (
	errNoAvailableBackends = errors.New("no backends are available")
)

func New(clientset *kubernetes.Clientset, namespace, service string) (pool.Pool, error) {
	return &serviceMonitor{
		client:      clientset,
		namespace:   namespace,
		service:     service,
		backendsmap: make(map[string]bool),
	}, nil
}

type serviceMonitor struct {
	client    *kubernetes.Clientset
	namespace string
	service   string

	lock         sync.Mutex
	backendsmap  map[string]bool
	backendslist []string
}

func (s *serviceMonitor) setAddresses(ep *corev1.Endpoints) {
	newmap := make(map[string]bool)
	newlist := make([]string, 0, 5)

	for sidx := range ep.Subsets {
		subset := &ep.Subsets[sidx]

		for aidx := range subset.Addresses {
			address := &subset.Addresses[aidx]

			newmap[address.IP] = true
			newlist = append(newlist, address.IP)
		}
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	s.backendsmap = newmap
	s.backendslist = newlist
}

func (s *serviceMonitor) getPool() (backends []string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.backendslist
}

func (s *serviceMonitor) Run(ctx context.Context) {
	for {
		err := s.run(ctx)

		select {
		case <-ctx.Done():
			return
		default:
		}

		if err != nil {
			log.Print(err)
		}

		time.Sleep(time.Second)
	}
}

func (s *serviceMonitor) run(ctx context.Context) error {
	endpointswatcher, err := s.client.CoreV1().Endpoints(s.namespace).Watch(ctx, metav1.SingleObject(metav1.ObjectMeta{
		Namespace: s.namespace,
		Name:      s.service,
	}))
	if err != nil {
		return err
	}

	for ev := range endpointswatcher.ResultChan() {
		if ep, ok := ev.Object.(*corev1.Endpoints); ok {
			s.setAddresses(ep)
		} else {
			log.Fatalf("Received event for invalid object: %+v", ev)
		}
	}

	return nil
}

// GetBackend returns a backend
func (s *serviceMonitor) GetBackend(ctx context.Context) (string, error) {
	backends := s.getPool()

	if len(backends) == 0 {
		return "", errNoAvailableBackends
	}

	return backends[rand.Intn(len(backends))], nil
}

// IsBackendAlive returns whether a given backend is available
func (s *serviceMonitor) IsBackendAlive(ctx context.Context, backend string) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.backendsmap[backend], nil
}
