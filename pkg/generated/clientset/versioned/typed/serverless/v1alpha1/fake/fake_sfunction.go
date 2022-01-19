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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/wm775825/sync-controller/pkg/apis/serverless/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSfunctions implements SfunctionInterface
type FakeSfunctions struct {
	Fake *FakeServerlessV1alpha1
	ns   string
}

var sfunctionsResource = schema.GroupVersionResource{Group: "serverless.wm775825.info", Version: "v1alpha1", Resource: "sfunctions"}

var sfunctionsKind = schema.GroupVersionKind{Group: "serverless.wm775825.info", Version: "v1alpha1", Kind: "Sfunction"}

// Get takes name of the sfunction, and returns the corresponding sfunction object, and an error if there is any.
func (c *FakeSfunctions) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Sfunction, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(sfunctionsResource, c.ns, name), &v1alpha1.Sfunction{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Sfunction), err
}

// List takes label and field selectors, and returns the list of Sfunctions that match those selectors.
func (c *FakeSfunctions) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.SfunctionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(sfunctionsResource, sfunctionsKind, c.ns, opts), &v1alpha1.SfunctionList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.SfunctionList{ListMeta: obj.(*v1alpha1.SfunctionList).ListMeta}
	for _, item := range obj.(*v1alpha1.SfunctionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested sfunctions.
func (c *FakeSfunctions) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(sfunctionsResource, c.ns, opts))

}

// Create takes the representation of a sfunction and creates it.  Returns the server's representation of the sfunction, and an error, if there is any.
func (c *FakeSfunctions) Create(ctx context.Context, sfunction *v1alpha1.Sfunction, opts v1.CreateOptions) (result *v1alpha1.Sfunction, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(sfunctionsResource, c.ns, sfunction), &v1alpha1.Sfunction{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Sfunction), err
}

// Update takes the representation of a sfunction and updates it. Returns the server's representation of the sfunction, and an error, if there is any.
func (c *FakeSfunctions) Update(ctx context.Context, sfunction *v1alpha1.Sfunction, opts v1.UpdateOptions) (result *v1alpha1.Sfunction, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(sfunctionsResource, c.ns, sfunction), &v1alpha1.Sfunction{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Sfunction), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeSfunctions) UpdateStatus(ctx context.Context, sfunction *v1alpha1.Sfunction, opts v1.UpdateOptions) (*v1alpha1.Sfunction, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(sfunctionsResource, "status", c.ns, sfunction), &v1alpha1.Sfunction{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Sfunction), err
}

// Delete takes name of the sfunction and deletes it. Returns an error if one occurs.
func (c *FakeSfunctions) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(sfunctionsResource, c.ns, name), &v1alpha1.Sfunction{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSfunctions) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(sfunctionsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.SfunctionList{})
	return err
}

// Patch applies the patch and returns the patched sfunction.
func (c *FakeSfunctions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Sfunction, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(sfunctionsResource, c.ns, name, pt, data, subresources...), &v1alpha1.Sfunction{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Sfunction), err
}
