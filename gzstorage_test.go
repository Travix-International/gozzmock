package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzStorage_NotInit_AddPanics(t *testing.T) {
	//Arrange
	var storage gzStorage

	//Assert
	assert.Panics(t, storage.add())
}

func TestGzStorage_NotInit_RemovePanics(t *testing.T) {
	//Arrange
	var storage gzStorage

	//Assert
	assert.Panics(t, storage.remove())
}

func TestGzStorage_NotInit_GetOrderedPanics(t *testing.T) {
	//Arrange
	var storage gzStorage

	//Assert
	assert.Panics(t, storage.getOrdered())
}

func TestGzStorage_Empty(t *testing.T) {
	//Arrange
	var storage gzStorage

	//Act
	storage.init()

	//Assert
	assert.Empty(t, res)
}

func TestGzStorage_EmptyGetOrdered(t *testing.T) {
	//Arrange
	var storage gzStorage

	//Act
	storage.init()
	res := storage.getOrdered()

	//Assert
	assert.Empty(t, res)
}

func TestGzStorage_Add(t *testing.T) {
	//Arrange
	var storage gzStorage
	storage.init()

	//Act
	storage.add("k", Expectation{Key: "key"})

	//Assert
	res := storage.getOrdered()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "key", res[0].Key)
}

func TestGzStorage_AddSameKey(t *testing.T) {
	//Arrange
	var storage gzStorage
	storage.init()

	//Act
	storage.add("k1", Expectation{Key: "v1"})
	storage.add("k1", Expectation{Key: "v2"})

	//Assert
	res := storage.getOrdered()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "v2", res[0].Key)
}

func TestGzStorage_Remove(t *testing.T) {
	//Arrange
	var storage gzStorage
	storage.init()
	storage.add("k", Expectation{Key: "key"})

	//Act
	storage.remove("k")

	//Assert
	res := storage.getOrdered()
	assert.NotNil(t, res)
	assert.Empty(t, res)
}

func TestGzStorage_RemoveWrongKey(t *testing.T) {
	//Arrange
	var storage gzStorage
	storage.init()
	storage.add("k", Expectation{Key: "key"})

	//Act
	storage.remove("kW")

	//Assert
	res := storage.getOrdered()
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "key", res[0].Key)
}

func TestGzStorage_Order(t *testing.T) {
	//Arrange
	var storage gzStorage
	storage.init()
	storage.add("p10", Expectation{Priority: 10})
	storage.add("p5", Expectation{Priority: 5})
	storage.add("p15", Expectation{Priority: 15})

	//Act
	res := storage.getOrdered()

	//Assert
	assert.Equal(t, 3, len(res))
	assert.Equal(t, "p5", res[0].Key)
	assert.Equal(t, "p10", res[1].Key)
	assert.Equal(t, "p15", res[2].Key)
}
