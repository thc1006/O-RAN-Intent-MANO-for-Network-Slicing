package mocks

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockK8sClient provides a mock implementation of client.Client for testing
type MockK8sClient struct {
	GetFunc    func(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error
	ListFunc   func(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
	CreateFunc func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
	UpdateFunc func(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
	DeleteFunc func(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
	PatchFunc  func(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error

	// Call tracking
	GetCalls    []GetCall
	CreateCalls []CreateCall
	UpdateCalls []UpdateCall
	DeleteCalls []DeleteCall
}

type GetCall struct {
	Key types.NamespacedName
	Obj client.Object
}

type CreateCall struct {
	Obj client.Object
}

type UpdateCall struct {
	Obj client.Object
}

type DeleteCall struct {
	Obj client.Object
}

func (m *MockK8sClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	m.GetCalls = append(m.GetCalls, GetCall{Key: key, Obj: obj})
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key, obj, opts...)
	}
	return errors.NewNotFound(schema.GroupResource{}, key.Name)
}

func (m *MockK8sClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, list, opts...)
	}
	return nil
}

func (m *MockK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	m.CreateCalls = append(m.CreateCalls, CreateCall{Obj: obj})
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, obj, opts...)
	}
	return nil
}

func (m *MockK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	m.UpdateCalls = append(m.UpdateCalls, UpdateCall{Obj: obj})
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, obj, opts...)
	}
	return nil
}

func (m *MockK8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	m.DeleteCalls = append(m.DeleteCalls, DeleteCall{Obj: obj})
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, obj, opts...)
	}
	return nil
}

func (m *MockK8sClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	if m.PatchFunc != nil {
		return m.PatchFunc(ctx, obj, patch, opts...)
	}
	return nil
}

func (m *MockK8sClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	return nil
}

func (m *MockK8sClient) Status() client.StatusWriter {
	return &MockStatusWriter{}
}

func (m *MockK8sClient) Scheme() *runtime.Scheme {
	return runtime.NewScheme()
}

func (m *MockK8sClient) RESTMapper() client.RESTMapper {
	return nil
}

type MockStatusWriter struct{}

func (m *MockStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.SubResourceUpdateOption) error {
	return nil
}

func (m *MockStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
	return nil
}