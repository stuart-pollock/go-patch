package patch

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type ReplaceOp struct {
	Path  Pointer
	Value interface{} // will be cloned using yaml library
}

type mutationCtx struct {
	PrevUpdate func(interface{})
	I          int
	Obj        interface{}
}

func replaceOpCloneValueErr(err error) error {
	return fmt.Errorf("ReplaceOp cloning value: %s", err)
}

func (op ReplaceOp) Apply(doc interface{}) (interface{}, error) {
	tokens := op.Path.Tokens()

	if len(tokens) == 1 {
		// Ensure that value is not modified by future operations
		clonedValue, err := op.cloneValue(op.Value)
		if err != nil {
			return nil, replaceOpCloneValueErr(err)
		}
		return clonedValue, nil
	}

	ctxStack := []*mutationCtx{&mutationCtx{
		PrevUpdate: func(newObj interface{}) { doc = newObj },
		I:          0,
		Obj:        doc,
	}}
	for len(ctxStack) != 0 {
		// Pop the next context off the stack
		ctx := ctxStack[len(ctxStack)-1]
		ctxStack = ctxStack[:len(ctxStack)-1]

		// Terminate if done
		if ctx.I+1 >= len(tokens) {
			continue
		}

		token := tokens[ctx.I+1]
		isLast := ctx.I == len(tokens)-2
		currPath := NewPointer(tokens[:ctx.I+2])

		switch typedToken := token.(type) {
		case IndexToken:
			typedObj, ok := ctx.Obj.([]interface{})
			if !ok {
				return nil, NewOpArrayMismatchTypeErr(currPath, ctx.Obj)
			}

			if isLast {
				clonedValue, err := op.cloneValue(op.Value)
				if err != nil {
					return nil, replaceOpCloneValueErr(err)
				}
				idx, err := ArrayInsertion{Index: typedToken.Index, Modifiers: typedToken.Modifiers, Array: typedObj, Path: currPath}.Concrete()
				if err != nil {
					return nil, err
				}
				ctx.PrevUpdate(idx.Update(typedObj, clonedValue))
			} else {
				idx, err := ArrayIndex{Index: typedToken.Index, Modifiers: typedToken.Modifiers, Array: typedObj, Path: currPath}.Concrete()
				if err != nil {
					return nil, err
				}
				ctxStack = append(ctxStack, &mutationCtx{
					PrevUpdate: func(newObj interface{}) { typedObj[idx] = newObj },
					I:          ctx.I + 1,
					Obj:        typedObj[idx],
				})
			}

		case AfterLastIndexToken:
			typedObj, ok := ctx.Obj.([]interface{})
			if !ok {
				return nil, NewOpArrayMismatchTypeErr(currPath, ctx.Obj)
			}

			if isLast {
				clonedValue, err := op.cloneValue(op.Value)
				if err != nil {
					return nil, replaceOpCloneValueErr(err)
				}
				ctx.PrevUpdate(append(typedObj, clonedValue))
			} else {
				return nil, fmt.Errorf("Expected after last index token to be last in path '%s'", op.Path)
			}

		case MatchingIndexToken:
			typedObj, ok := ctx.Obj.([]interface{})
			if !ok {
				return nil, NewOpArrayMismatchTypeErr(currPath, ctx.Obj)
			}

			var idxs []int

			for itemIdx, item := range typedObj {
				typedItem, ok := item.(map[interface{}]interface{})
				if ok {
					if typedItem[typedToken.Key] == typedToken.Value {
						idxs = append(idxs, itemIdx)
					}
				}
			}

			if typedToken.Optional && len(idxs) == 0 {
				if isLast {
					clonedValue, err := op.cloneValue(op.Value)
					if err != nil {
						return nil, replaceOpCloneValueErr(err)
					}
					ctx.PrevUpdate(append(typedObj, clonedValue))
				} else {
					o := map[interface{}]interface{}{typedToken.Key: typedToken.Value}
					ctx.PrevUpdate(append(typedObj, o))
					ctxStack = append(ctxStack, &mutationCtx{
						PrevUpdate: ctx.PrevUpdate, // no need to change prevUpdate since matching item can only be a map
						I:          ctx.I + 1,
						Obj:        o,
					})
				}
			} else {
				if len(idxs) != 1 {
					return nil, OpMultipleMatchingIndexErr{currPath, idxs}
				}

				if isLast {
					clonedValue, err := op.cloneValue(op.Value)
					idx, err := ArrayInsertion{Index: idxs[0], Modifiers: typedToken.Modifiers, Array: typedObj, Path: currPath}.Concrete()
					if err != nil {
						return nil, err
					}

					ctx.PrevUpdate(idx.Update(typedObj, clonedValue))
				} else {
					idx, err := ArrayIndex{Index: idxs[0], Modifiers: typedToken.Modifiers, Array: typedObj, Path: currPath}.Concrete()
					if err != nil {
						return nil, err
					}

					// no need to change prevUpdate since matching item can only be a map
					ctxStack = append(ctxStack, &mutationCtx{
						PrevUpdate: ctx.PrevUpdate, // no need to change prevUpdate since matching item can only be a map
						I:          ctx.I + 1,
						Obj:        typedObj[idx],
					})
				}
			}

		case KeyToken:
			typedObj, ok := ctx.Obj.(map[interface{}]interface{})
			if !ok {
				return nil, NewOpMapMismatchTypeErr(currPath, ctx.Obj)
			}

			o, found := typedObj[typedToken.Key]
			if !found && !typedToken.Optional {
				return nil, OpMissingMapKeyErr{typedToken.Key, currPath, typedObj}
			}

			if isLast {
				clonedValue, err := op.cloneValue(op.Value)
				if err != nil {
					return nil, replaceOpCloneValueErr(err)
				}
				typedObj[typedToken.Key] = clonedValue
			} else {
				if !found {
					// Determine what type of value to create based on next token
					switch tokens[ctx.I+2].(type) {
					case AfterLastIndexToken:
						o = []interface{}{}
					case WildcardToken:
						o = []interface{}{}
					case MatchingIndexToken:
						o = []interface{}{}
					case KeyToken:
						o = map[interface{}]interface{}{}
					default:
						errMsg := "Expected to find key, matching index or after last index token at path '%s'"
						return nil, fmt.Errorf(errMsg, NewPointer(tokens[:ctx.I+3]))
					}

					typedObj[typedToken.Key] = o
				}

				ctxStack = append(ctxStack, &mutationCtx{
					PrevUpdate: func(newObj interface{}) { typedObj[typedToken.Key] = newObj },
					I:          ctx.I + 1,
					Obj:        o,
				})
			}

		case WildcardToken:
			if isLast {
				return nil, fmt.Errorf("Wildcard must not be the last token")
			}

			typedObj, ok := ctx.Obj.([]interface{})
			if !ok {
				return nil, NewOpArrayMismatchTypeErr(currPath, ctx.Obj)
			}

			for idx, o := range typedObj {
				ctxStack = append(ctxStack, &mutationCtx{
					PrevUpdate: func(newObj interface{}) { typedObj[idx] = newObj },
					I:          ctx.I + 1,
					Obj:        o,
				})
			}

		default:
			return nil, OpUnexpectedTokenErr{token, currPath}
		}
	}

	return doc, nil
}

func (ReplaceOp) cloneValue(in interface{}) (out interface{}, err error) {
	defer func() {
		if recoverVal := recover(); recoverVal != nil {
			err = fmt.Errorf("Recovered: %s", recoverVal)
		}
	}()

	bytes, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(bytes, &out)
	if err != nil {
		return nil, err
	}

	return out, nil
}
