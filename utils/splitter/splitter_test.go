package splitter

import (
	"reflect"
	"testing"
)

func TestNewSplitter(t *testing.T) {
	data := []string{"a", "b", "c", "d", "e", "f"}
	splitCount := 2
	splitter := NewSplitter(splitCount, data)
	if splitter.splitCount != splitCount {
		t.Errorf("Expected splitCount %d, got %d", splitCount, splitter.splitCount)
	}
	if !reflect.DeepEqual(splitter.data, data) {
		t.Errorf("Expected data %v, got %v", data, splitter.data)
	}
}

func TestGetSplitResult_EvenSplit(t *testing.T) {
	data := []string{"a", "b", "c", "d"}
	splitCount := 2
	splitter := NewSplitter(splitCount, data)
	expected := []*SplitResult{
		{Item: []string{"a", "b"}},
		{Item: []string{"c", "d"}},
	}
	result := splitter.GetSplitResult()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %v, got %v", expected, result)
	}
}

func TestGetSplitResult_OddSplit(t *testing.T) {
	data := []string{"a", "b", "c", "d", "e"}
	splitCount := 2
	splitter := NewSplitter(splitCount, data)
	expected := []*SplitResult{
		{Item: []string{"a", "b"}},
		{Item: []string{"c", "d"}},
		{Item: []string{"e"}},
	}
	result := splitter.GetSplitResult()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %v, got %v", expected, result)
	}
}

func TestGetSplitResult_SingleElement(t *testing.T) {
	data := []string{"a"}
	splitCount := 2
	splitter := NewSplitter(splitCount, data)
	expected := []*SplitResult{{Item: []string{"a"}}}
	result := splitter.GetSplitResult()
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected result %v, got %v", expected, result)
	}
}
