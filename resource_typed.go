package rest

// TypedResource 是编译期类型安全的 HTTP 资源包装, 内部委托给动态核心 Resource。
type TypedResource[T any] struct {
	*Resource
	model *TypedModel[T] // 仅用于 ModelValue(), 阴影嵌入的同名字段
}

func NewTypedResource[T any](model *TypedModel[T], cfg ResourceConfig) *TypedResource[T] {
	return &TypedResource[T]{
		Resource: NewResource(model.Model, cfg),
		model:    model,
	}
}

func NewTypedResourceWithOptions[T any](cfg ResourceConfig, opts ...Option) (resource *TypedResource[T], err error) {
	var modelValue *TypedModel[T]
	modelValue, err = NewTypedModel[T](opts...)
	if err != nil {
		return
	}
	resource = NewTypedResource(modelValue, cfg)
	resource.Register()
	return
}

func (r *TypedResource[T]) ModelValue() *TypedModel[T] {
	return r.model
}
