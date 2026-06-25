package recognition

import "sync/atomic"

type FaceCache struct {
	vectors atomic.Value
}

func NewFaceCache(initial []StudentVector) *FaceCache {
	cache := &FaceCache{}
	cache.Replace(initial)
	return cache
}

func (c *FaceCache) Replace(next []StudentVector) {
	copied := make([]StudentVector, len(next))
	copy(copied, next)
	c.vectors.Store(copied)
}

func (c *FaceCache) All() []StudentVector {
	value := c.vectors.Load()
	if value == nil {
		return nil
	}
	vectors := value.([]StudentVector)
	copied := make([]StudentVector, len(vectors))
	copy(copied, vectors)
	return copied
}

func (c *FaceCache) Students() []StudentVector {
	all := c.All()
	students := make([]StudentVector, 0, len(all))
	for _, vector := range all {
		if vector.PersonType == "" || vector.PersonType == "student" {
			students = append(students, vector)
		}
	}
	return students
}

func (c *FaceCache) Size() int {
	value := c.vectors.Load()
	if value == nil {
		return 0
	}
	return len(value.([]StudentVector))
}
