/*
Copyright 2020 Red Hat, Inc.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	versioned "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned"
	internalinterfaces "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/listers/operators/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ClusterServiceVersionInformer provides access to a shared informer and lister for
// ClusterServiceVersions.
type ClusterServiceVersionInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.ClusterServiceVersionLister
}

type clusterServiceVersionInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewClusterServiceVersionInformer constructs a new informer for ClusterServiceVersion type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewClusterServiceVersionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredClusterServiceVersionInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredClusterServiceVersionInformer constructs a new informer for ClusterServiceVersion type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredClusterServiceVersionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OperatorsV1alpha1().ClusterServiceVersions(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.OperatorsV1alpha1().ClusterServiceVersions(namespace).Watch(options)
			},
		},
		&operatorsv1alpha1.ClusterServiceVersion{},
		resyncPeriod,
		indexers,
	)
}

func (f *clusterServiceVersionInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredClusterServiceVersionInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *clusterServiceVersionInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&operatorsv1alpha1.ClusterServiceVersion{}, f.defaultInformer)
}

func (f *clusterServiceVersionInformer) Lister() v1alpha1.ClusterServiceVersionLister {
	return v1alpha1.NewClusterServiceVersionLister(f.Informer().GetIndexer())
}
