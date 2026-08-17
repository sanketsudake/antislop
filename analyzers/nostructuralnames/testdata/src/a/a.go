package a

import geoshapes "geo"

// --- invalid ---

type UserShape struct { // want `nostructuralnames: name "UserShape" describes structure \("shape"\) rather than the domain role; rename it for what it represents`
	Shape string // want `nostructuralnames: name "Shape" describes structure \("shape"\)`
}

func (u UserShape) ShapeOf() string { return u.Shape } // want `nostructuralnames: name "ShapeOf" describes structure \("shape"\)`

func reshape(n int) int { return n } // want `nostructuralnames: name "reshape" describes structure \("shape"\)`

func area(shape int) int { return shape } // want `nostructuralnames: name "shape" describes structure \("shape"\)`

func sides() (shapeCount int) { return 0 } // want `nostructuralnames: name "shapeCount" describes structure \("shape"\)`

var shapeCache = map[string]int{} // want `nostructuralnames: name "shapeCache" describes structure \("shape"\)`

const ShapeLimit = 10 // want `nostructuralnames: name "ShapeLimit" describes structure \("shape"\)`

type reader interface {
	ShapeName() string // want `nostructuralnames: name "ShapeName" describes structure \("shape"\)`
}

func locals() int {
	shapes := 1 // want `nostructuralnames: name "shapes" describes structure \("shape"\)`
	return shapes
}

func labelled() int {
	total := 0
shapeLoop: // want `nostructuralnames: name "shapeLoop" describes structure \("shape"\)`
	for i := 0; i < 3; i++ {
		if i == 1 {
			break shapeLoop
		}
		total += i
	}
	return total
}

func first[Shape any](items []Shape) Shape { return items[0] } // want `nostructuralnames: name "Shape" describes structure \("shape"\)`

type box struct{ n int }

func (shape box) size() int { return shape.n } // want `nostructuralnames: name "shape" describes structure \("shape"\)`

func ranged(items []int) int {
	total := 0
	for _, shapeItem := range items { // want `nostructuralnames: name "shapeItem" describes structure \("shape"\)`
		total += shapeItem
	}
	return total
}

// --- valid ---

// Names that state the domain role carry no structural term.
type Order struct {
	ID string
}

func (o Order) Total() int { return 0 }

func count(orders []Order) int { return len(orders) }

// A symbol declared in another package is that package's naming decision, and
// so is the name this file uses to reach it.
func external(s geoshapes.Shape) string { return s.Name() }

// The blank identifier declares nothing to rename.
func discard(items []int) int {
	_, n := 0, len(items)
	return n
}

var _ = reader(nil)
