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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"time"

	v1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	scheme "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CatalogSourcesGetter has a method to return a CatalogSourceInterface.
// A group's client should implement this interface.
type CatalogSourcesGetter interface {
	CatalogSources(namespace string) CatalogSourceInterface
}

// CatalogSourceInterface has methods to work with CatalogSource resources.
type CatalogSourceInterface interface {
	Create(*v1alpha1.CatalogSource) (*v1alpha1.CatalogSource, error)
	Update(*v1alpha1.CatalogSource) (*v1alpha1.CatalogSource, error)
	UpdateStatus(*v1alpha1.CatalogSource) (*v1alpha1.CatalogSource, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.CatalogSource, error)
	List(opts v1.ListOptions) (*v1alpha1.CatalogSourceList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.CatalogSource, err error)
	CatalogSourceExpansion
}

// catalogSources implements CatalogSourceInterface
type catalogSources struct {
	client rest.Interface
	ns     string
}

// newCatalogSources returns a CatalogSources
func newCatalogSources(c *OperatorsV1alpha1Client, namespace string) *catalogSources {
	return &catalogSources{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the catalogSource, and returns the corresponding catalogSource object, and an error if there is any.
func (c *catalogSources) Get(name string, options v1.GetOptions) (result *v1alpha1.CatalogSource, err error) {
	result = &v1alpha1.CatalogSource{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("catalogsources").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CatalogSources that match those selectors.
func (c *catalogSources) List(opts v1.ListOptions) (result *v1alpha1.CatalogSourceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.CatalogSourceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("catalogsources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested catalogSources.
func (c *catalogSources) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("catalogsources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a catalogSource and creates it.  Returns the server's representation of the catalogSource, and an error, if there is any.
func (c *catalogSources) Create(catalogSource *v1alpha1.CatalogSource) (result *v1alpha1.CatalogSource, err error) {
	result = &v1alpha1.CatalogSource{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("catalogsources").
		Body(catalogSource).
		Do().
		Into(result)
	return
}

// Update takes the representation of a catalogSource and updates it. Returns the server's representation of the catalogSource, and an error, if there is any.
func (c *catalogSources) Update(catalogSource *v1alpha1.CatalogSource) (result *v1alpha1.CatalogSource, err error) {
	result = &v1alpha1.CatalogSource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("catalogsources").
		Name(catalogSource.Name).
		Body(catalogSource).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *catalogSources) UpdateStatus(catalogSource *v1alpha1.CatalogSource) (result *v1alpha1.CatalogSource, err error) {
	result = &v1alpha1.CatalogSource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("catalogsources").
		Name(catalogSource.Name).
		SubResource("status").
		Body(catalogSource).
		Do().
		Into(result)
	return
}

// Delete takes name of the catalogSource and deletes it. Returns an error if one occurs.
func (c *catalogSources) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("catalogsources").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *catalogSources) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("catalogsources").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched catalogSource.
func (c *catalogSources) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.CatalogSource, err error) {
	result = &v1alpha1.CatalogSource{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("catalogsources").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
