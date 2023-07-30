package main

import (
	"sync"
)

// Collection это sync.Map-ка, хранящая в себе произвольные итемы.
type Collection struct {
	items sync.Map
	close chan struct{}
}

// Item представляет собой интерфейс для хранения произвольных данных.
type item struct {
	data interface{}
}

// NewCollection создаёт инстанс коллекции.
func NewCollection() *Collection {
	c := &Collection{ //nolint:exhaustruct
		close: make(chan struct{}),
	}

	return c
}

// (collection *Collection) Get(key interface{}) (interface{}, bool) достаёт данные по заданному ключу из коллекции.
func (collection *Collection) Get(key interface{}) (interface{}, bool) {
	obj, exists := collection.items.Load(key)

	if !exists {
		return nil, false
	}

	item := obj.(item)

	return item.data, true
}

// (collection *Collection) Set(key interface{}, value interface{}) сохраняет данные с заданным ключом в коллекцию.
func (collection *Collection) Set(key interface{}, value interface{}) {
	collection.items.Store(key, item{
		data: value,
	})
}

// (collection *Collection) Range(f func(key, value interface{}) bool) применяет функцию f ко всем ключам в коллекции.
func (collection *Collection) Range(f func(key, value interface{}) bool) {
	fn := func(key, value interface{}) bool {
		item := value.(item)

		return f(key, item.data)
	}

	collection.items.Range(fn)
}

// (collection *Collection) Delete(key interface{}) удаляет ключ и значение из коллекции данных.
func (collection *Collection) Delete(key interface{}) {
	collection.items.Delete(key)
}

// (collection *Collection) Close() очищает и высвобождает ресурсы, занятые коллекцией.
func (collection *Collection) Close() {
	collection.close <- struct{}{}
	collection.items = sync.Map{}
}

/* vim: set ft=go noet ai ts=4 sw=4 sts=4: */
