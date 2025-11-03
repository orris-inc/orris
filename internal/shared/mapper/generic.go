package mapper

import "reflect"

type Mapper[T any, D any] struct {
	toDTO    func(T) D
	toDomain func(D) T
}

func New[T any, D any](toDTO func(T) D, toDomain func(D) T) *Mapper[T, D] {
	return &Mapper[T, D]{
		toDTO:    toDTO,
		toDomain: toDomain,
	}
}

func (m *Mapper[T, D]) ToDTO(entity T) D {
	return m.toDTO(entity)
}

func (m *Mapper[T, D]) ToDomain(dto D) T {
	return m.toDomain(dto)
}

func (m *Mapper[T, D]) ToDTOList(entities []T) []D {
	if entities == nil {
		return nil
	}

	dtos := make([]D, 0, len(entities))
	for _, entity := range entities {
		dtos = append(dtos, m.toDTO(entity))
	}
	return dtos
}

func (m *Mapper[T, D]) ToDomainList(dtos []D) []T {
	if dtos == nil {
		return nil
	}

	entities := make([]T, 0, len(dtos))
	for _, dto := range dtos {
		entities = append(entities, m.toDomain(dto))
	}
	return entities
}

func MapSlice[T any, R any](items []T, mapFunc func(T) R) []R {
	if items == nil {
		return nil
	}

	result := make([]R, 0, len(items))
	for _, item := range items {
		result = append(result, mapFunc(item))
	}
	return result
}

func MapSlicePtr[T any, R any](items []*T, mapFunc func(*T) *R) []*R {
	if items == nil {
		return nil
	}

	result := make([]*R, 0, len(items))
	for _, item := range items {
		if item != nil {
			result = append(result, mapFunc(item))
		}
	}
	return result
}

func CopyFields(src interface{}, dst interface{}) {
	srcVal := reflect.ValueOf(src)
	dstVal := reflect.ValueOf(dst)

	if srcVal.Kind() == reflect.Ptr {
		srcVal = srcVal.Elem()
	}
	if dstVal.Kind() == reflect.Ptr {
		dstVal = dstVal.Elem()
	}

	if !dstVal.CanSet() {
		return
	}

	srcType := srcVal.Type()
	dstType := dstVal.Type()

	for i := 0; i < srcType.NumField(); i++ {
		srcField := srcType.Field(i)
		srcFieldValue := srcVal.Field(i)

		if dstField, ok := dstType.FieldByName(srcField.Name); ok {
			if dstField.Type == srcField.Type {
				dstFieldValue := dstVal.FieldByName(srcField.Name)
				if dstFieldValue.CanSet() {
					dstFieldValue.Set(srcFieldValue)
				}
			}
		}
	}
}
