package patch

type MoveOp struct {
	Path Pointer
	From Pointer
}

func (op MoveOp) Apply(doc interface{}) (interface{}, error) {
	val, err := FindOp{Path: op.From}.Apply(doc)
	if err != nil {
		return nil, err
	}

	doc, err = ReplaceOp{Path: op.Path, Value: val}.Apply(doc)
	if err != nil {
		return nil, err
	}

	doc, err = RemoveOp{Path: op.From}.Apply(doc)
	if err != nil {
		return nil, err
	}

	return doc, nil
}
