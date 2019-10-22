package patch_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/stuart-pollock/go-patch/patch"
)

var _ = Describe("MoveOp.Apply", func() {
	It("returns an error if path is for the entire document", func() {
		_, err := MoveOp{Path: MustNewPointerFromString(""), From: MustNewPointerFromString("")}.Apply("a")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Cannot remove entire document"))
	})

	It("returns an error if from does not exist", func() {
		doc := map[interface{}]interface{}{
			"xyz": map[interface{}]interface{}{
				"nested": "blah",
			},
		}

		_, err := MoveOp{Path: MustNewPointerFromString("/xyz/new_nested?"), From: MustNewPointerFromString("/xyz/new_nested")}.Apply(doc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to find a map key 'new_nested' for path '/xyz/new_nested' (found map keys: 'nested')"))
	})

	It("returns an error if path is not permissible", func() {
		doc := map[interface{}]interface{}{
			"xyz": "xyz",
		}

		_, err := MoveOp{Path: MustNewPointerFromString("/abc/def/ghi"), From: MustNewPointerFromString("/xyz")}.Apply(doc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected to find a map key 'abc' for path '/abc' (found map keys: 'xyz')"))
	})

	It("returns an error if from is not permissible", func() {
		doc := map[interface{}]interface{}{
			"xyz": "xyz",
		}

		_, err := MoveOp{Path: MustNewPointerFromString("/abc?"), From: MustNewPointerFromString("")}.Apply(doc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Cannot remove entire document"))
	})

	It("moves matching item within a map", func() {
		doc := map[interface{}]interface{}{
			"xyz": "xyz",
		}

		res, err := MoveOp{Path: MustNewPointerFromString("/abc?"), From: MustNewPointerFromString("/xyz")}.Apply(doc)
		Expect(err).ToNot(HaveOccurred())

		Expect(res).To(Equal(map[interface{}]interface{}{
			"abc": "xyz",
		}))
	})

	It("moves matching item into a map", func() {
		doc := map[interface{}]interface{}{
			"abc": map[interface{}]interface{}{
				"def": "def",
			},
			"xyz": "xyz",
		}

		res, err := MoveOp{Path: MustNewPointerFromString("/abc/xyz?"), From: MustNewPointerFromString("/xyz")}.Apply(doc)
		Expect(err).ToNot(HaveOccurred())

		Expect(res).To(Equal(map[interface{}]interface{}{
			"abc": map[interface{}]interface{}{
				"def": "def",
				"xyz": "xyz",
			},
		}))
	})

	It("moves matching item out of a map", func() {
		doc := map[interface{}]interface{}{
			"abc": map[interface{}]interface{}{
				"def": "def",
			},
			"xyz": "xyz",
		}

		res, err := MoveOp{Path: MustNewPointerFromString("/def?"), From: MustNewPointerFromString("/abc/def")}.Apply(doc)
		Expect(err).ToNot(HaveOccurred())

		Expect(res).To(Equal(map[interface{}]interface{}{
			"abc": map[interface{}]interface{}{},
			"def": "def",
			"xyz": "xyz",
		}))
	})
})
