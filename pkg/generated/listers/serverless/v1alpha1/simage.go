/*
Copyright The Kubernetes Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/wm775825/sync-controller/pkg/apis/serverless/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// SimageLister helps list Simages.
type SimageLister interface {
	// List lists all Simages in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.Simage, err error)
	// Simages returns an object that can list and get Simages.
	Simages(namespace string) SimageNamespaceLister
	SimageListerExpansion
}

// simageLister implements the SimageLister interface.
type simageLister struct {
	indexer cache.Indexer
}

// NewSimageLister returns a new SimageLister.
func NewSimageLister(indexer cache.Indexer) SimageLister {
	return &simageLister{indexer: indexer}
}

// List lists all Simages in the indexer.
func (s *simageLister) List(selector labels.Selector) (ret []*v1alpha1.Simage, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Simage))
	})
	return ret, err
}

// Simages returns an object that can list and get Simages.
func (s *simageLister) Simages(namespace string) SimageNamespaceLister {
	return simageNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// SimageNamespaceLister helps list and get Simages.
type SimageNamespaceLister interface {
	// List lists all Simages in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.Simage, err error)
	// Get retrieves the Simage from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.Simage, error)
	SimageNamespaceListerExpansion
}

// simageNamespaceLister implements the SimageNamespaceLister
// interface.
type simageNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Simages in the indexer for a given namespace.
func (s simageNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Simage, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Simage))
	})
	return ret, err
}

// Get retrieves the Simage from the indexer for a given namespace and name.
func (s simageNamespaceLister) Get(name string) (*v1alpha1.Simage, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("simage"), name)
	}
	return obj.(*v1alpha1.Simage), nil
}
