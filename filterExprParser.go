// Copyright 2018-2019 Couchbase, Inc. All rights reserved.

package gojsonsm

import (
	"fmt"
	"github.com/alecthomas/participle"
	"strings"
)

// EBNF Grammar describing the parser

// FilterExpression	= ( OpenParen ComboCondition CloseParen) | AndCondition { "OR" AndCondition }

// FilterExpression			= AndCondition { "OR" AndCondition }
// AndCondition 			= Condition { "AND" Condition }
// Condition				= ( [ "NOT" ] Condition ) | Operand
// Operand					= { Paren } BooleanExpr | ( LHS ( CheckOp | ( CompareOp RHS) ) ) { Paren }
// Paren			    	= OpenParen | CloseParen
// BooleanExpr				= Boolean | BooleanFuncExpr
// LHS 						= ConstFuncExpr | Field | Value
// RHS 						= ConstFuncExpr | Value | Field
// CompareOp				= "=" | "<>" | ">" | ">=" | "<" | "<="
// CheckOp					= "EXISTS" | ( "IS" [ "NOT" ] ( NULL | MISSING ) )
// Field					= OnePath { "." OnePath }
// OnePath					= ( PathFuncExpression | @String | @Ident ){ ArrayIndex }
// ArrayIndex				= "[" [ @"-" ] @Int "]"
// Value					= @String
// ConstFuncExpr			= ConstFuncNoArg | ConstFuncOneArg | ConstFuncTwoArgs
// ConstFuncNoArg			= ConstFuncNoArgName "(" ")"
// ConstFuncNoArgName 		= "PI" | "E"
// ConstFuncOneArg 			= ConstFuncOneArgName "(" ConstFuncArgument ")"
// ConstFuncOneArgName  	= "ABS" | "ACOS"...
// ConstFuncTwoArgs			= ConstFuncTwoArgsName "(" ConstFuncArgument "," ConstFuncArgument ")"
// ConstFuncTwoArgsName 	 = "ATAN2" | "POW"
// ConstFuncArgument		= Field | Value | ConstFuncExpr
// ConstFuncArgumentRHS		= Value
// PathFuncExpression		= OnePathFuncNoArg
// OnePathFuncNoArg			= OnePathFuncNoArgName "(" ")"
// OnePathFuncNoArgName		= "META"
// BooleanFuncExpr			= BooleanFuncTwoArgs
// BooleanFuncTwoArgs		= BooleanFuncTwoArgsName "(" ConstFuncArgument "," ConstFuncArgumentRHS ")"
// BooleanFuncTwoArgsName 	= "REGEXP_CONTAINS"

type FEComboExpression struct {
	AndConditions []*FEAndCondition `@@ { "OR" @@ }`
}

type FilterExpression struct {
	//	OpenParens    []*FEOpenParen      `{ @@ }`
	AndConditions []*FEAndCondition `( @@ { "OR" @@ } )`
	//	CloseParens   []*FECloseParen     `{ @@ }`
	SubFilterExpr []*FilterExpression `{ "AND" @@ }`
	stHelper      StepThroughIface
}

func (fe *FilterExpression) String() string {
	output := []string{}

	first := true
	for _, expr := range fe.AndConditions {
		if first {
			first = false
		} else {
			output = append(output, "OR")
		}
		output = append(output, expr.String())
	}

	for _, expr := range fe.SubFilterExpr {
		if first {
			first = false
		} else {
			output = append(output, "AND")
		}
		output = append(output, expr.String())
	}

	return strings.Join(output, " ")
}

type filterExpressionST struct {
	fe *FilterExpression

	i int
}

func (f *filterExpressionST) IsTerm() bool {
	return false
}

func (f *filterExpressionST) Init() error {
	f.i = 0
	for _, oneCond := range f.fe.AndConditions {
		oneCond.stHelper = &feAndConditionST{e: oneCond}
		if err := oneCond.stHelper.Init(); err != nil {
			return err
		}
	}
	return nil
}

func (f *filterExpressionST) Done() bool {
	return f.fe.AndConditions[len(f.fe.AndConditions)-1].stHelper.Done()
}

type FEOpenParen struct {
	Parens   string `@"("`
	stHelper StepThroughIface
}

func (feop *FEOpenParen) String() string {
	return "("
}

type feOpenParenST struct {
	e      *FEOpenParen
	called bool
}

func (f *feOpenParenST) Init() error {
	if f.e == nil {
		return ErrorNotFound
	}
	return nil
}

func (f *feOpenParenST) IsTerm() bool {
	return true
}

func (f *feOpenParenST) Done() bool {
	return f.called
}

type FECloseParen struct {
	Parens   string `@")"`
	stHelper StepThroughIface
}

func (fecp *FECloseParen) String() string {
	return ")"
}

type feCloseParenST struct {
	e      *FECloseParen
	called bool
}

func (f *feCloseParenST) Init() error {
	if f.e == nil {
		return ErrorNotFound
	}
	return nil
}

func (f *feCloseParenST) IsTerm() bool {
	return true
}

func (f *feCloseParenST) Done() bool {
	return f.called
}

type FEAndCondition struct {
	OpenParens   []*FEOpenParen  `{ @@ }`
	OrConditions []*FECondition  `@@ { "AND" @@ }`
	CloseParens  []*FECloseParen `{ @@ }`
	stHelper     StepThroughIface
}

func (ac *FEAndCondition) String() string {
	output := []string{}

	for _, e := range ac.OpenParens {
		output = append(output, e.String())
	}

	first := true
	for _, e := range ac.OrConditions {
		if first {
			first = false
		} else {
			output = append(output, "AND")
		}
		output = append(output, e.String())
	}

	for _, e := range ac.CloseParens {
		output = append(output, e.String())
	}

	return strings.Join(output, " ")
}

type feAndConditionST struct {
	e *FEAndCondition

	i int
}

func (f *feAndConditionST) IsTerm() bool {
	return false
}

func (f *feAndConditionST) Done() bool {
	return f.e.OrConditions[len(f.e.OrConditions)-1].stHelper.Done()
}

func (f *feAndConditionST) Init() error {
	f.i = 0

	for _, oneCond := range f.e.OrConditions {
		oneCond.stHelper = &feConditionST{e: oneCond}
		if err := oneCond.stHelper.Init(); err != nil {
			return err
		}
	}

	return nil
}

type FECondition struct {
	//	PreParen  []*FEOpenParen  `{ @@ }`
	PreParen []*FEOpenParen
	Not      *FECondition      `"NOT" @@`
	Operand  *FEOperand        `| @@`
	SubExpr  *FilterExpression `| @@`
	//	PostParen []*FECloseParen `{ @@ }`
	PostParen []*FECloseParen
	stHelper  StepThroughIface
}

func (fec *FECondition) String() string {
	var outputStr []string

	for i := 0; i < len(fec.PreParen); i++ {
		outputStr = append(outputStr, fec.PreParen[i].String())
	}

	if fec.Not != nil {
		outputStr = append(outputStr, fmt.Sprintf("NOT %v", fec.Not.String()))
	} else if fec.Operand != nil {
		outputStr = append(outputStr, fec.Operand.String())
	} else if fec.SubExpr != nil {
		outputStr = append(outputStr, fec.SubExpr.String())
	} else {
		outputStr = append(outputStr, "?? (FECondition)")
	}

	for i := 0; i < len(fec.PostParen); i++ {
		outputStr = append(outputStr, fec.PostParen[i].String())
	}

	return strings.Join(outputStr, " ")
}

type feConditionST struct {
	e                   *FECondition
	skipPreParen        bool
	skipPostParen       bool
	needToGetSpecialNot bool
	hasGottenSpecialNot bool
}

func (f *feConditionST) IsTerm() bool {
	return false
}

func (f *feConditionST) Done() bool {
	if !f.skipPostParen {
		return f.e.PostParen[len(f.e.PostParen)-1].stHelper.Done()
	} else if f.needToGetSpecialNot && f.hasGottenSpecialNot {
		return f.e.Not.stHelper.Done()
	} else {
		return f.e.Operand.stHelper.Done()
	}
}

func (f *feConditionST) Init() error {
	if f.e.Not == nil && f.e.Operand == nil {
		return ErrorNotFound
	}

	if len(f.e.PreParen) > 0 {
		for _, p := range f.e.PreParen {
			p.stHelper = &feOpenParenST{e: p}
			err := p.stHelper.Init()
			if err != nil {
				return err
			}
		}
	} else {
		f.skipPreParen = true
	}

	if f.e.Not != nil {
		f.needToGetSpecialNot = true
		f.e.Not.stHelper = &feConditionST{e: f.e.Not}
		err := f.e.Not.stHelper.Init()
		if err != nil {
			return err
		}
	}

	if f.e.Operand != nil {
		f.e.Operand.stHelper = &feOperandST{e: f.e.Operand}
		err := f.e.Operand.stHelper.Init()
		if err != nil {
			return err
		}
	}

	if len(f.e.PostParen) > 0 {
		for _, p := range f.e.PostParen {
			p.stHelper = &feCloseParenST{e: p}
			err := p.stHelper.Init()
			if err != nil {
				return err
			}
		}
	} else {
		f.skipPostParen = true
	}

	return nil
}

type FEOperand struct {
	BooleanExpr *FEBooleanExpr `@@ |`
	LHS         *FELhs         `( @@ (`
	Op          *FECompareOp   `( @@`
	RHS         *FERhs         `@@ ) | `
	CheckOp     *FECheckOp     `@@ ) )`
	stHelper    StepThroughIface
}

func (feo *FEOperand) String() string {
	if feo.BooleanExpr != nil {
		return feo.BooleanExpr.String()
	} else if feo.CheckOp != nil {
		return fmt.Sprintf("%v %v", feo.LHS.String(), feo.CheckOp.String())
	} else if feo.Op != nil {
		return fmt.Sprintf("%v %v %v", feo.LHS.String(), feo.Op.String(), feo.RHS.String())
	} else {
		return "?? (FEOperand)"
	}
}

type feOperandST struct {
	e *FEOperand
}

func (f *feOperandST) Init() error {
	if f.e.BooleanExpr != nil {
		f.e.BooleanExpr.stHelper = &feBooleanExprST{e: f.e.BooleanExpr}
		return f.e.BooleanExpr.stHelper.Init()
	} else if f.e.LHS != nil {
		f.e.LHS.stHelper = &feLhsST{e: f.e.LHS}
		err := f.e.LHS.stHelper.Init()
		if err != nil {
			return err
		}
		if f.e.Op != nil {
			f.e.Op.stHelper = &feCompareOpST{e: f.e.Op}
			err := f.e.Op.stHelper.Init()
			if err != nil {
				return err
			}

			if f.e.RHS == nil {
				return ErrorNotFound
			}

			f.e.RHS.stHelper = &feRhsST{e: f.e.RHS}
			return f.e.RHS.stHelper.Init()
		} else if f.e.CheckOp != nil {
			f.e.CheckOp.stHelper = &feCheckOpST{e: f.e.CheckOp}
			return f.e.CheckOp.stHelper.Init()
		} else {
			return ErrorNotFound
		}
	} else {
		return ErrorNotFound
	}
}

func (f *feOperandST) IsTerm() bool {
	return false
}

func (f *feOperandST) Done() bool {
	if f.e.BooleanExpr != nil {
		return f.e.BooleanExpr.stHelper.Done()
	} else if f.e.LHS != nil {
		if f.e.CheckOp != nil {
			return f.e.LHS.stHelper.Done() && f.e.CheckOp.stHelper.Done()
		} else {
			return f.e.LHS.stHelper.Done() && f.e.Op.stHelper.Done() && f.e.RHS.stHelper.Done()
		}
	} else {
		return false
	}
}

type FEBooleanExpr struct {
	BooleanVal  *FEBoolean         `@@ |`
	BooleanFunc *FEBooleanFuncExpr `@@`
	stHelper    StepThroughIface
}

func (be *FEBooleanExpr) String() string {
	if be.BooleanVal != nil {
		return be.BooleanVal.String()
	} else if be.BooleanFunc != nil {
		return be.BooleanFunc.String()
	} else {
		return "?? (FEBooleanExpr)"
	}
}

type feBooleanExprST struct {
	e *FEBooleanExpr
}

func (f *feBooleanExprST) IsTerm() bool {
	if f.e.BooleanFunc != nil {
		return f.e.BooleanFunc.stHelper.IsTerm()
	} else {
		return f.e.BooleanVal.stHelper.IsTerm()
	}
}

func (f *feBooleanExprST) Done() bool {
	if f.e.BooleanFunc != nil {
		return f.e.BooleanFunc.stHelper.Done()
	} else {
		return f.e.BooleanVal.stHelper.Done()
	}
}

func (f *feBooleanExprST) Init() error {
	if f.e.BooleanFunc != nil {
		f.e.BooleanFunc.stHelper = &feBooleanFuncExprST{e: f.e.BooleanFunc}
		return f.e.BooleanFunc.stHelper.Init()
	} else {
		f.e.BooleanVal.stHelper = &feBooleanST{e: f.e.BooleanVal}
		return f.e.BooleanVal.stHelper.Init()
	}
	return nil
}

type FEBoolean struct {
	TVal     *bool `@"TRUE" |`
	TVal1    *bool `@"true" |`
	FVal     *bool `@"FALSE" |`
	FVal1    *bool `@"false"`
	stHelper StepThroughIface
}

func (feb *FEBoolean) String() string {
	if feb.TVal != nil && *feb.TVal == true {
		return "TRUE(bool)"
	} else if feb.TVal1 != nil && *feb.TVal1 == true {
		return "true(bool)"
	} else if feb.FVal != nil && *feb.FVal == true {
		return "FALSE(bool)"
	} else if feb.FVal1 != nil && *feb.FVal1 == true {
		return "false(bool)"
	}
	return ""
}

func (feb *FEBoolean) GetBool() bool {
	if feb.TVal != nil && *feb.TVal == true {
		return true
	} else if feb.TVal1 != nil && *feb.TVal1 == true {
		return true
	} else if feb.FVal != nil && *feb.FVal == true {
		return false
	} else if feb.FVal1 != nil && *feb.FVal1 == true {
		return false
	}
	return false
}

func (feb *FEBoolean) IsSet() bool {
	return feb.TVal != nil || feb.TVal1 != nil || feb.FVal != nil || feb.FVal1 != nil
}

type feBooleanST struct {
	e                *FEBoolean
	hasBeenRetrieved bool
}

func (f *feBooleanST) IsTerm() bool {
	return true
}

func (f *feBooleanST) Done() bool {
	return f.hasBeenRetrieved
}

func (f *feBooleanST) Init() error {
	if f.e == nil {
		return ErrorNotFound
	}
	f.hasBeenRetrieved = false
	return nil
}

type FELhs struct {
	Func     *FEConstFuncExpression `( @@ |`
	Field    *FEField               `@@ |`
	Value    *FEValue               `@@ )`
	stHelper StepThroughIface
}

func (fel *FELhs) String() string {
	if fel.Field != nil {
		return fel.Field.String()
	} else if fel.Value != nil {
		return fel.Value.String()
	} else if fel.Func != nil {
		return fel.Func.String()
	} else {
		return "?? (FELhs)"
	}
}

type feLhsST struct {
	e *FELhs
}

func (f *feLhsST) Init() error {
	if f.e.Field != nil {
		f.e.Field.stHelper = &feFieldST{e: f.e.Field}
		return f.e.Field.stHelper.Init()
	} else if f.e.Value != nil {
		f.e.Value.stHelper = &feValueST{e: f.e.Value}
		return f.e.Value.stHelper.Init()
	} else if f.e.Func != nil {
		f.e.Func.stHelper = &feConstFuncExpressionST{e: f.e.Func}
		return f.e.Func.stHelper.Init()
	} else {
		return ErrorNotFound
	}
}

func (f *feLhsST) IsTerm() bool {
	return false
}

func (f *feLhsST) Done() bool {
	if f.e.Field != nil {
		return f.e.Field.stHelper.Done()
	} else if f.e.Value != nil {
		return f.e.Value.stHelper.Done()
	} else if f.e.Func != nil {
		return f.e.Func.stHelper.Done()
	} else {
		return false
	}
}

// Normally users do values on the RHS
type FERhs struct {
	Func     *FEConstFuncExpression `( @@ |`
	Value    *FEValue               `@@ |`
	Field    *FEField               `@@ )`
	stHelper StepThroughIface
}

func (fer *FERhs) String() string {
	if fer.Field != nil {
		return fer.Field.String()
	} else if fer.Value != nil {
		return fer.Value.String()
	} else if fer.Func != nil {
		return fer.Func.String()
	} else {
		return "??"
	}
}

type feRhsST struct {
	e *FERhs
}

func (f *feRhsST) Init() error {
	if f.e.Field != nil {
		f.e.Field.stHelper = &feFieldST{e: f.e.Field}
		return f.e.Field.stHelper.Init()
	} else if f.e.Value != nil {
		f.e.Value.stHelper = &feValueST{e: f.e.Value}
		return f.e.Value.stHelper.Init()
	} else if f.e.Func != nil {
		f.e.Func.stHelper = &feConstFuncExpressionST{e: f.e.Func}
		return f.e.Func.stHelper.Init()
	} else {
		return ErrorNotFound
	}
}

func (f *feRhsST) IsTerm() bool {
	return false
}

func (f *feRhsST) Done() bool {
	if f.e.Field != nil {
		return f.e.Field.stHelper.Done()
	} else if f.e.Value != nil {
		return f.e.Value.stHelper.Done()
	} else if f.e.Func != nil {
		return f.e.Func.stHelper.Done()
	} else {
		return false
	}
}

type FEField struct {
	Path     []*FEOnePath `@@ { "." @@ }`
	stHelper StepThroughIface
}

func (fef *FEField) String() string {
	output := []string{}
	for _, onePath := range fef.Path {
		output = append(output, onePath.String())
	}
	return strings.Join(output, ".")
}

type feFieldST struct {
	e   *FEField
	idx int
}

func (f *feFieldST) Init() error {
	if len(f.e.Path) == 0 {
		return ErrorNotFound
	}

	for _, path := range f.e.Path {
		path.stHelper = &feOnePathST{e: path}
		err := path.stHelper.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *feFieldST) IsTerm() bool {
	return false
}

func (f *feFieldST) Done() bool {
	return f.e.Path[len(f.e.Path)-1].stHelper.Done()
}

type FEOnePath struct {
	OnePathFunc   *FEOnePathFuncExpr `( @@  |`
	EscapedStrVal string             "@String  |"
	StrValue      string             "@Ident )"
	ArrayIndexes  []*FEArrayIndex    `{ @@ }`
	stHelper      StepThroughIface
}

func (feop *FEOnePath) String() string {
	output := []string{}
	if feop.OnePathFunc != nil {
		output = append(output, feop.OnePathFunc.String())
	} else if len(feop.StrValue) > 0 {
		output = append(output, feop.StrValue)
	} else if len(feop.EscapedStrVal) > 0 {
		output = append(output, feop.EscapedStrVal)
	} else {
		output = append(output, "")
	}
	for i := 0; i < len(feop.ArrayIndexes); i++ {
		output = append(output, feop.ArrayIndexes[i].String())
	}
	return strings.Join(output, " ")
}

type feOnePathST struct {
	e            *FEOnePath
	skipArrayIdx bool
	terminal     bool
	called       bool
	idx          int
}

func (f *feOnePathST) Init() error {
	if len(f.e.EscapedStrVal) == 0 && f.e.OnePathFunc == nil && len(f.e.StrValue) == 0 {
		return ErrorNotFound
	}

	if f.e.OnePathFunc != nil {
		f.e.OnePathFunc.stHelper = &feOnePathFuncExprST{e: f.e.OnePathFunc}
		err := f.e.OnePathFunc.stHelper.Init()
		if err != nil {
			return err
		}
	}

	if len(f.e.ArrayIndexes) == 0 {
		f.skipArrayIdx = true
	} else {
		for _, idx := range f.e.ArrayIndexes {
			idx.stHelper = &feArrayIndexST{e: idx}
			err := idx.stHelper.Init()
			if err != nil {
				return err
			}
		}
	}

	if len(f.e.ArrayIndexes) == 0 && f.e.OnePathFunc == nil {
		f.terminal = true
	}

	return nil
}

func (f *feOnePathST) IsTerm() bool {
	return f.terminal
}

func (f *feOnePathST) Done() bool {
	if f.skipArrayIdx {
		if f.e.OnePathFunc == nil {
			return f.called
		} else {
			return f.e.OnePathFunc.stHelper.Done()
		}
	} else {
		return f.e.ArrayIndexes[len(f.e.ArrayIndexes)-1].stHelper.Done()
	}
}

type FEArrayIndex struct {
	ArrayIndex string `"[" [ @"-" ] @Int "]"`
	stHelper   StepThroughIface
}

func (i *FEArrayIndex) String() string {
	return fmt.Sprintf("[%v]", i.ArrayIndex)
}

type feArrayIndexST struct {
	e      *FEArrayIndex
	called bool
}

func (f *feArrayIndexST) Init() error {
	if len(f.e.ArrayIndex) == 0 {
		return ErrorNotFound
	}
	return nil
}

func (f *feArrayIndexST) IsTerm() bool {
	return true
}

func (f *feArrayIndexST) Done() bool {
	return f.called
}

type FEOnePathFuncExpr struct {
	OnePathFuncNoArg *FEOnePathFuncNoArg `@@`
	stHelper         StepThroughIface
}

func (e *FEOnePathFuncExpr) String() string {
	if e.OnePathFuncNoArg != nil {
		return e.OnePathFuncNoArg.String()
	} else {
		return "?? FEOnePathFuncExpr"
	}
}

type feOnePathFuncExprST struct {
	e *FEOnePathFuncExpr
}

func (f *feOnePathFuncExprST) Init() error {
	if f.e.OnePathFuncNoArg == nil {
		return ErrorNotFound
	}
	f.e.OnePathFuncNoArg.stHelper = &feOnePathFuncNoArgST{e: f.e.OnePathFuncNoArg}
	return f.e.OnePathFuncNoArg.stHelper.Init()
}

func (f *feOnePathFuncExprST) IsTerm() bool {
	return false
}

func (f *feOnePathFuncExprST) Done() bool {
	return f.e.OnePathFuncNoArg.stHelper.Done()
}

type FEOnePathFuncNoArg struct {
	OnePathFuncNoArgName *FEOnePathFuncNoArgName `( @@ "(" ")" )`
	stHelper             StepThroughIface
}

func (na *FEOnePathFuncNoArg) String() string {
	if na.OnePathFuncNoArgName != nil {
		return fmt.Sprintf("%v()", na.OnePathFuncNoArgName.String())
	} else {
		return "?? (FEOnePathFuncNoArg)"
	}
}

type feOnePathFuncNoArgST struct {
	e *FEOnePathFuncNoArg
}

func (f *feOnePathFuncNoArgST) Init() error {
	if f.e.OnePathFuncNoArgName == nil {
		return ErrorNotFound
	}
	f.e.OnePathFuncNoArgName.stHelper = &feOnePathFuncNoArgNameST{e: f.e.OnePathFuncNoArgName}
	return f.e.OnePathFuncNoArgName.stHelper.Init()
}

func (f *feOnePathFuncNoArgST) IsTerm() bool {
	return false
}

func (f *feOnePathFuncNoArgST) Done() bool {
	return f.e.OnePathFuncNoArgName.stHelper.Done()
}

type FEOnePathFuncNoArgName struct {
	Meta     *bool `@"META"`
	stHelper StepThroughIface
}

func (n *FEOnePathFuncNoArgName) String() string {
	if n.Meta != nil && *n.Meta == true {
		return "META"
	} else {
		return "?? (FEOnePathFuncNoArgName)"
	}
}

type feOnePathFuncNoArgNameST struct {
	e      *FEOnePathFuncNoArgName
	called bool
}

func (f *feOnePathFuncNoArgNameST) Init() error {
	if f.e.Meta == nil {
		return ErrorNotFound
	}
	return nil
}

func (f *feOnePathFuncNoArgNameST) IsTerm() bool {
	return true
}

func (f *feOnePathFuncNoArgNameST) Done() bool {
	return f.called
}

type FEValue struct {
	StrValue   *string  `@String |`
	IntValue   *int     `@Int |`
	FloatValue *float64 `@Float`
	stHelper   StepThroughIface
}

func (fev *FEValue) String() string {
	if fev.StrValue != nil {
		return *fev.StrValue
	} else if fev.IntValue != nil {
		return fmt.Sprintf("%v", *fev.IntValue)
	} else if fev.FloatValue != nil {
		return fmt.Sprintf("%v", *fev.FloatValue)
	} else {
		return "??"
	}
}

type feValueST struct {
	e      *FEValue
	called bool
}

func (f *feValueST) Init() error {
	if f.e.StrValue == nil && f.e.IntValue == nil && f.e.FloatValue == nil {
		return ErrorNotFound
	}
	return nil
}

func (f *feValueST) IsTerm() bool {
	return true
}

func (f *feValueST) Done() bool {
	return f.called
}

// We have to do this funky way of matching because our FEOperand expression may not be composed of a compareOp
// And due to the complicated FEOperand op, we have to do char by char match so we can catch the not-matched case
// and go to the other type of operands

type FEOpChar struct {
	Equal       *bool `( @"=" |`
	LessThan    *bool `@"<" |`
	GreaterThan *bool `@">" )`
	stHelper    StepThroughIface
}

type feOpCharST struct {
	e      *FEOpChar
	called bool
}

func (f *feOpCharST) Init() error {
	if f.e.Equal == nil && f.e.LessThan == nil && f.e.GreaterThan == nil {
		return ErrorNotFound
	}
	return nil
}

func (f *feOpCharST) IsTerm() bool {
	return true
}

func (f *feOpCharST) Done() bool {
	return f.called
}

type FECompareOp struct {
	OpChars0 *FEOpChar `@@`
	OpChars1 *FEOpChar `[ @@ ]`
	stHelper StepThroughIface
}

func (feo *FECompareOp) IsEqual() bool {
	// =
	return feo.OpChars0 != nil && feo.OpChars0.Equal != nil && feo.OpChars0 != nil && feo.OpChars1 == nil
}

func (feo *FECompareOp) IsNotEqual() bool {
	// <>
	return feo.OpChars0 != nil && feo.OpChars0.LessThan != nil && feo.OpChars1 != nil && feo.OpChars1.GreaterThan != nil
}

func (feo *FECompareOp) IsGreaterThan() bool {
	// >
	return feo.OpChars0 != nil && feo.OpChars0.GreaterThan != nil && feo.OpChars1 == nil
}

func (feo *FECompareOp) IsGreaterThanOrEqualTo() bool {
	// >=
	return feo.OpChars0 != nil && feo.OpChars0.GreaterThan != nil && feo.OpChars1 != nil && feo.OpChars1.Equal != nil
}

func (feo *FECompareOp) IsLessThan() bool {
	// <
	return feo.OpChars0 != nil && feo.OpChars0.LessThan != nil && feo.OpChars1 == nil
}

func (feo *FECompareOp) IsLessThanOrEqualTo() bool {
	// <=
	return feo.OpChars0 != nil && feo.OpChars0.LessThan != nil && feo.OpChars1 != nil && feo.OpChars1.Equal != nil
}

func (feo *FECompareOp) String() string {
	if feo.IsEqual() {
		return "="
	} else if feo.IsNotEqual() {
		return "!="
	} else if feo.IsGreaterThan() {
		return ">"
	} else if feo.IsGreaterThanOrEqualTo() {
		return ">="
	} else if feo.IsLessThan() {
		return "<"
	} else if feo.IsLessThanOrEqualTo() {
		return "<="
	}
	return ""
}

type feCompareOpST struct {
	e *FECompareOp
}

func (f *feCompareOpST) Init() error {
	if f.e.OpChars0 == nil {
		return ErrorNotFound
	} else {
		f.e.OpChars0.stHelper = &feOpCharST{e: f.e.OpChars0}
		err := f.e.OpChars0.stHelper.Init()
		if err != nil {
			return err
		}
	}

	if f.e.OpChars1 != nil {
		f.e.OpChars1.stHelper = &feOpCharST{e: f.e.OpChars1}
		return f.e.OpChars1.stHelper.Init()
	}
	return nil
}

func (f *feCompareOpST) IsTerm() bool {
	return false
}

func (f *feCompareOpST) Done() bool {
	if f.e.OpChars1 != nil {
		return f.e.OpChars0.stHelper.Done() && f.e.OpChars1.stHelper.Done()
	} else {
		return f.e.OpChars0.stHelper.Done()
	}
}

type FECheckOp struct {
	Exists   *bool `@"EXISTS" | ( "IS"`
	Not      *bool `[ @"NOT" ]`
	Null     *bool `( @"NULL" |`
	Missing  *bool `@"MISSING" ) )`
	stHelper StepThroughIface
}

func (feco *FECheckOp) IsExists() bool {
	return feco.Exists != nil && *feco.Exists == true
}

func (feco *FECheckOp) isNot() bool {
	return feco.Not != nil && *feco.Not == true
}

func (feco *FECheckOp) IsMissing() bool {
	return !feco.isNot() && feco.Missing != nil && *feco.Missing == true
}

func (feco *FECheckOp) IsNotMissing() bool {
	return !feco.IsMissing()
}

func (feco *FECheckOp) IsNull() bool {
	return !feco.isNot() && feco.Null != nil && *feco.Null == true
}

func (feco *FECheckOp) IsNotNull() bool {
	return !feco.IsNull()
}

func (feco *FECheckOp) String() string {
	if feco.IsExists() {
		return "EXISTS"
	} else if feco.IsMissing() {
		return "IS MISSING"
	} else if feco.IsNotMissing() {
		return "IS NOT MISSING"
	} else if feco.IsNull() {
		return "IS NULL"
	} else if feco.IsNotNull() {
		return "IS NOT NULL"
	} else {
		return "?? (FECheckOp)"
	}
}

type feCheckOpST struct {
	e      *FECheckOp
	called bool
}

func (f *feCheckOpST) Init() error {
	if f.e.Exists == nil && f.e.Not == nil && f.e.Null == nil && f.e.Missing == nil {
		return ErrorNotFound
	}
	return nil
}

func (f *feCheckOpST) IsTerm() bool {
	return true
}

func (f *feCheckOpST) Done() bool {
	return f.called
}

// Technically we could have an slice of arguments, but having OneArg vs NoArg vs TwoArg could
// allow us to do more strict function check (i.e. certain funcs should only allow one argument, etc, at this level)
type FEConstFuncExpression struct {
	ConstFuncNoArg   *FEConstFuncNoArg   `@@ |`
	ConstFuncOneArg  *FEConstFuncOneArg  `@@ |`
	ConstFuncTwoArgs *FEConstFuncTwoArgs `@@`
	stHelper         StepThroughIface
}

func (f *FEConstFuncExpression) String() string {
	if f.ConstFuncNoArg != nil {
		return f.ConstFuncNoArg.String()
	} else if f.ConstFuncOneArg != nil {
		return f.ConstFuncOneArg.String()
	} else if f.ConstFuncTwoArgs != nil {
		return f.ConstFuncTwoArgs.String()
	} else {
		return "?? (FEConstFuncExpression)"
	}
}

type feConstFuncExpressionST struct {
	e *FEConstFuncExpression
}

func (f *feConstFuncExpressionST) Init() error {
	if f.e.ConstFuncNoArg != nil {
		f.e.ConstFuncNoArg.stHelper = &feConstFuncNoArgST{e: f.e.ConstFuncNoArg}
		return f.e.ConstFuncNoArg.stHelper.Init()
	} else if f.e.ConstFuncOneArg != nil {
		f.e.ConstFuncOneArg.stHelper = &feConstFuncOneArgST{e: f.e.ConstFuncOneArg}
		return f.e.ConstFuncOneArg.stHelper.Init()
	} else if f.e.ConstFuncTwoArgs != nil {
		f.e.ConstFuncTwoArgs.stHelper = &feConstFuncTwoArgsST{e: f.e.ConstFuncTwoArgs}
		return f.e.ConstFuncTwoArgs.stHelper.Init()
	} else {
		return ErrorNotFound
	}
}

func (f *feConstFuncExpressionST) Done() bool {
	if f.e.ConstFuncNoArg != nil {
		return f.e.ConstFuncNoArg.stHelper.Done()
	} else if f.e.ConstFuncOneArg != nil {
		return f.e.ConstFuncOneArg.stHelper.Done()
	} else {
		return f.e.ConstFuncTwoArgs.stHelper.Done()
	}
}

func (f *feConstFuncExpressionST) IsTerm() bool {
	return false
}

type FEConstFuncNoArg struct {
	ConstFuncNoArgName *FEConstFuncNoArgName `( @@ "(" ")" )`
	stHelper           StepThroughIface
}

func (f *FEConstFuncNoArg) String() string {
	if f.ConstFuncNoArgName != nil {
		return fmt.Sprintf("%v()", f.ConstFuncNoArgName.String())
	} else {
		return "?? (FEConstFuncNoArg)"
	}
}

type feConstFuncNoArgST struct {
	e *FEConstFuncNoArg
}

func (f *feConstFuncNoArgST) Init() error {
	if f.e.ConstFuncNoArgName != nil {
		f.e.ConstFuncNoArgName.stHelper = &feConstFuncNoArgNameST{e: f.e.ConstFuncNoArgName}
		return f.e.ConstFuncNoArgName.stHelper.Init()
	}
	return ErrorNotFound
}

func (f *feConstFuncNoArgST) IsTerm() bool {
	return false
}

func (f *feConstFuncNoArgST) Done() bool {
	return f.e.ConstFuncNoArgName.stHelper.Done()
}

type FEConstFuncNoArgName struct {
	Pi       *bool `@"PI" |` // FuncPi
	E        *bool `@"E"`    // FuncE
	stHelper StepThroughIface
}

func (n *FEConstFuncNoArgName) String() string {
	if n.E != nil && *n.E == true {
		return "E"
	} else if n.Pi != nil && *n.Pi == true {
		return "PI"
	} else {
		return "?? (FEConstFuncNoArgName)"
	}
}

type feConstFuncNoArgNameST struct {
	e      *FEConstFuncNoArgName
	called bool
}

func (f *feConstFuncNoArgNameST) Init() error {
	if f.e.E == nil && f.e.Pi == nil {
		return ErrorNotFound
	}
	return nil
}

func (f *feConstFuncNoArgNameST) IsTerm() bool {
	return true
}

func (f *feConstFuncNoArgNameST) Done() bool {
	return f.called
}

// Order matters
type FEConstFuncArgument struct {
	SubFunc  *FEConstFuncExpression `@@ |`
	Field    *FEField               `@@ |`
	Argument *FEValue               `@@`
	stHelper StepThroughIface
}

func (arg *FEConstFuncArgument) String() string {
	if arg.Argument != nil {
		return arg.Argument.String()
	} else if arg.SubFunc != nil {
		return arg.SubFunc.String()
	} else if arg.Field != nil {
		return arg.Field.String()
	} else {
		return "?? (FEConstFuncArgument)"
	}
}

type feConstFuncArgumentST struct {
	e *FEConstFuncArgument
}

func (f *feConstFuncArgumentST) Init() error {
	if f.e.SubFunc != nil {
		f.e.SubFunc.stHelper = &feConstFuncExpressionST{e: f.e.SubFunc}
		return f.e.SubFunc.stHelper.Init()
	} else if f.e.Field != nil {
		f.e.Field.stHelper = &feFieldST{e: f.e.Field}
		return f.e.Field.stHelper.Init()
	} else if f.e.Argument != nil {
		f.e.Argument.stHelper = &feValueST{e: f.e.Argument}
		return f.e.Argument.stHelper.Init()
	} else {
		return ErrorNotFound
	}
}

func (f *feConstFuncArgumentST) IsTerm() bool {
	return false
}

func (f *feConstFuncArgumentST) Done() bool {
	if f.e.SubFunc != nil {
		return f.e.SubFunc.stHelper.Done()
	} else if f.e.Field != nil {
		return f.e.Field.stHelper.Done()
	} else if f.e.Argument != nil {
		return f.e.Argument.stHelper.Done()
	} else {
		return true
	}
}

// Prioritize value over field
type FEConstFuncArgumentRHS struct {
	SubFunc  *FEConstFuncExpression `@@ |`
	Argument *FEValue               `@@`
	stHelper StepThroughIface
}

func (arg *FEConstFuncArgumentRHS) String() string {
	if arg.Argument != nil {
		return arg.Argument.String()
	} else if arg.SubFunc != nil {
		return arg.SubFunc.String()
	} else {
		return "?? (FEConstFuncArgument)"
	}
}

type feConstFuncArgumentRHSST struct {
	e *FEConstFuncArgumentRHS
}

func (f *feConstFuncArgumentRHSST) Init() error {
	if f.e.SubFunc != nil {
		f.e.SubFunc.stHelper = &feConstFuncExpressionST{e: f.e.SubFunc}
		return f.e.SubFunc.stHelper.Init()
	} else if f.e.Argument != nil {
		f.e.Argument.stHelper = &feValueST{e: f.e.Argument}
		return f.e.Argument.stHelper.Init()
	} else {
		return ErrorNotFound
	}
}

func (f *feConstFuncArgumentRHSST) IsTerm() bool {
	return false
}

func (f *feConstFuncArgumentRHSST) Done() bool {
	if f.e.SubFunc != nil {
		return f.e.SubFunc.stHelper.Done()
	} else {
		return f.e.Argument.stHelper.Done()
	}
}

type FEConstFuncOneArg struct {
	ConstFuncOneArgName *FEConstFuncOneArgName `( @@ "("`
	Argument            *FEConstFuncArgument   `@@ ")" )`
	stHelper            StepThroughIface
}

func (oa *FEConstFuncOneArg) String() string {
	return fmt.Sprintf("%v( %v )", oa.ConstFuncOneArgName.String(), oa.Argument.String())
}

type feConstFuncOneArgST struct {
	e *FEConstFuncOneArg
}

func (f *feConstFuncOneArgST) Init() error {
	if f.e.ConstFuncOneArgName == nil || f.e.Argument == nil {
		return ErrorNotFound
	}
	f.e.ConstFuncOneArgName.stHelper = &feConstFuncOneArgNameST{e: f.e.ConstFuncOneArgName}
	err := f.e.ConstFuncOneArgName.stHelper.Init()
	if err != nil {
		return err
	}
	f.e.Argument.stHelper = &feConstFuncArgumentST{e: f.e.Argument}
	return f.e.Argument.stHelper.Init()
}

func (f *feConstFuncOneArgST) IsTerm() bool {
	return false
}

func (f *feConstFuncOneArgST) Done() bool {
	return f.e.ConstFuncOneArgName.stHelper.Done() && f.e.Argument.stHelper.Done()
}

type FEConstFuncOneArgName struct {
	Abs      *bool `@"ABS" |`
	Acos     *bool `@"ACOS" |`
	Asin     *bool `@"ASIN" |`
	Atan     *bool `@"ATAN" |`
	Ceil     *bool `@"CEIL" |`
	Cos      *bool `@"COS" |`
	Date     *bool `@"DATE" |`
	Degrees  *bool `@"DEGREES" |`
	Exp      *bool `@"EXP" |`
	Floor    *bool `@"FLOOR" |`
	Log      *bool `@"LOG" |`
	Ln       *bool `@"LN" |`
	Sine     *bool `@"SIN" |`
	Tangent  *bool `@"TAN" |`
	Radians  *bool `@"RADIANS" |`
	Round    *bool `@"ROUND" |`
	Sqrt     *bool `@"SQRT"`
	stHelper StepThroughIface
}

func (arg *FEConstFuncOneArgName) String() string {
	if arg.Abs != nil && *arg.Abs == true {
		return "ABS"
	} else if arg.Acos != nil && *arg.Acos == true {
		return "ACOS"
	} else if arg.Asin != nil && *arg.Asin == true {
		return "ASIN"
	} else if arg.Atan != nil && *arg.Atan == true {
		return "ATAN"
	} else if arg.Ceil != nil && *arg.Ceil == true {
		return "CEIL"
	} else if arg.Cos != nil && *arg.Cos == true {
		return "COS"
	} else if arg.Date != nil && *arg.Date == true {
		return "DATE"
	} else if arg.Degrees != nil && *arg.Degrees == true {
		return "DEGREES"
	} else if arg.Exp != nil && *arg.Exp == true {
		return "EXP"
	} else if arg.Floor != nil && *arg.Floor == true {
		return "FLOOR"
	} else if arg.Log != nil && *arg.Log == true {
		return "LOG"
	} else if arg.Ln != nil && *arg.Ln == true {
		return "LN"
	} else if arg.Sine != nil && *arg.Sine == true {
		return "SIN"
	} else if arg.Tangent != nil && *arg.Tangent == true {
		return "TAN"
	} else if arg.Radians != nil && *arg.Radians == true {
		return "RADIANS"
	} else if arg.Round != nil && *arg.Round == true {
		return "ROUND"
	} else if arg.Sqrt != nil && *arg.Sqrt == true {
		return "SQRT"
	} else {
		return "?? (FEConstFuncOneArgName)"
	}
}

type feConstFuncOneArgNameST struct {
	e      *FEConstFuncOneArgName
	called bool
}

func (f *feConstFuncOneArgNameST) Init() error {
	if strings.Contains(f.e.String(), "??") {
		return ErrorNotFound
	}
	return nil
}

func (f *feConstFuncOneArgNameST) IsTerm() bool {
	return true
}

func (f *feConstFuncOneArgNameST) Done() bool {
	return f.called
}

type FEConstFuncTwoArgs struct {
	ConstFuncTwoArgsName *FEConstFuncTwoArgsName `( @@ "("`
	Argument0            *FEConstFuncArgument    `@@ "," `
	Argument1            *FEConstFuncArgument    `@@ ")" )`
	stHelper             StepThroughIface
}

func (fta *FEConstFuncTwoArgs) String() string {
	return fmt.Sprintf("%v( %v , %v )", fta.ConstFuncTwoArgsName.String(), fta.Argument0.String(), fta.Argument1.String())
}

type feConstFuncTwoArgsST struct {
	e *FEConstFuncTwoArgs
}

func (f *feConstFuncTwoArgsST) Init() error {
	if f.e.ConstFuncTwoArgsName == nil || f.e.Argument0 == nil || f.e.Argument1 == nil {
		return ErrorNotFound
	}

	f.e.ConstFuncTwoArgsName.stHelper = &feConstFuncTwoArgsNameST{e: f.e.ConstFuncTwoArgsName}
	err := f.e.ConstFuncTwoArgsName.stHelper.Init()
	if err != nil {
		return err
	}

	f.e.Argument0.stHelper = &feConstFuncArgumentST{e: f.e.Argument0}
	err = f.e.Argument0.stHelper.Init()
	if err != nil {
		return err
	}

	f.e.Argument1.stHelper = &feConstFuncArgumentST{e: f.e.Argument1}
	err = f.e.Argument1.stHelper.Init()
	return err
}

func (f *feConstFuncTwoArgsST) IsTerm() bool {
	return false
}

func (f *feConstFuncTwoArgsST) Done() bool {
	return f.e.ConstFuncTwoArgsName.stHelper.Done() && f.e.Argument0.stHelper.Done() && f.e.Argument1.stHelper.Done()
}

type FEConstFuncTwoArgsName struct {
	Atan2    *bool `@"ATAN2" |`
	Power    *bool `@"POW"`
	stHelper StepThroughIface
}

func (arg *FEConstFuncTwoArgsName) String() string {
	if arg.Atan2 != nil && *arg.Atan2 == true {
		return "ATAN2"
	} else if arg.Power != nil && *arg.Power == true {
		return "POW"
	} else {
		return "?? (FEConstFuncTwoArgsName)"
	}
}

type feConstFuncTwoArgsNameST struct {
	e      *FEConstFuncTwoArgsName
	called bool
}

func (f *feConstFuncTwoArgsNameST) Init() error {
	if f.e.Atan2 == nil && f.e.Power == nil {
		return ErrorNotFound
	}
	return nil
}

func (f *feConstFuncTwoArgsNameST) IsTerm() bool {
	return true
}

func (f *feConstFuncTwoArgsNameST) Done() bool {
	return f.called
}

type FEBooleanFuncExpr struct {
	BooleanFuncTwoArgs *FEBooleanFuncTwoArgs `@@`
	stHelper           StepThroughIface
}

func (e *FEBooleanFuncExpr) String() string {
	if e.BooleanFuncTwoArgs != nil {
		return e.BooleanFuncTwoArgs.String()
	} else {
		return "?? (FEBooleanFuncExpr)"
	}
}

type feBooleanFuncExprST struct {
	e *FEBooleanFuncExpr
}

func (f *feBooleanFuncExprST) Init() error {
	if f.e.BooleanFuncTwoArgs != nil {
		f.e.BooleanFuncTwoArgs.stHelper = &feBooleanFuncTwoArgsST{e: f.e.BooleanFuncTwoArgs}
		return f.e.BooleanFuncTwoArgs.stHelper.Init()
	}
	return ErrorNotFound
}

func (f *feBooleanFuncExprST) IsTerm() bool {
	return false
}

func (f *feBooleanFuncExprST) Done() bool {
	return f.e.BooleanFuncTwoArgs.stHelper.Done()
}

type FEBooleanFuncTwoArgs struct {
	BooleanFuncTwoArgsName *FEBooleanFuncTwoArgsName `( @@ "("`
	Argument0              *FEConstFuncArgument      `@@ ","`
	Argument1              *FEConstFuncArgumentRHS   `@@ ")" )`
	stHelper               StepThroughIface
}

func (a *FEBooleanFuncTwoArgs) String() string {
	if a.BooleanFuncTwoArgsName != nil {
		return fmt.Sprintf("%v( %v , %v )", a.BooleanFuncTwoArgsName.String(), a.Argument0.String(), a.Argument1.String())
	} else {
		return "?? (FEBooleanFuncTwoArgs)"
	}
}

type feBooleanFuncTwoArgsST struct {
	e *FEBooleanFuncTwoArgs
}

func (f *feBooleanFuncTwoArgsST) Init() error {
	if f.e.Argument0 == nil || f.e.Argument1 == nil || f.e.BooleanFuncTwoArgsName == nil {
		return ErrorNotFound
	}

	f.e.BooleanFuncTwoArgsName.stHelper = &feBooleanFuncTwoArgsNameST{e: f.e.BooleanFuncTwoArgsName}
	err := f.e.BooleanFuncTwoArgsName.stHelper.Init()
	if err != nil {
		return err
	}

	f.e.Argument0.stHelper = &feConstFuncArgumentST{e: f.e.Argument0}
	err = f.e.Argument0.stHelper.Init()
	if err != nil {
		return err
	}
	f.e.Argument1.stHelper = &feConstFuncArgumentRHSST{e: f.e.Argument1}
	return f.e.Argument1.stHelper.Init()
}

func (f *feBooleanFuncTwoArgsST) Done() bool {
	return f.e.stHelper.Done()
}

func (f *feBooleanFuncTwoArgsST) IsTerm() bool {
	return false
}

type FEBooleanFuncTwoArgsName struct {
	RegexContains *bool `@"REGEXP_CONTAINS"`
	stHelper      StepThroughIface
}

func (n *FEBooleanFuncTwoArgsName) String() string {
	if n.RegexContains != nil && *n.RegexContains == true {
		return "REGEXP_CONTAINS"
	} else {
		return "?? (FEBooleanFuncTwoArgsName)"
	}
}

type feBooleanFuncTwoArgsNameST struct {
	e      *FEBooleanFuncTwoArgsName
	called bool
}

func (f *feBooleanFuncTwoArgsNameST) Init() error {
	if f.e == nil {
		return ErrorNotFound
	}
	return nil
}

func (f *feBooleanFuncTwoArgsNameST) IsTerm() bool {
	return true
}

func (f *feBooleanFuncTwoArgsNameST) Done() bool {
	return f.called
}

func NewFilterExpressionParser(expression string) (*participle.Parser, *FilterExpression, error) {
	fe := &FilterExpression{}
	parser, err := participle.Build(fe)
	if err != nil || len(expression) == 0 {
		return parser, fe, err
	}

	err = parser.ParseString(expression, fe)

	return parser, fe, err
}

type FEConverter struct {
	expression *FilterExpression
}

func (c *FEConverter) Init() error {
	c.expression.stHelper = &filterExpressionST{fe: c.expression}
	return c.expression.stHelper.Init()
}

func (c *FEConverter) ConvertToAST() {

}

type StepThroughIface interface {
	Done() bool // Whether or not this level is all done
	Init() error
	IsTerm() bool // Whether or not this level is a terminal level
	//	Step() (string, interface{}) // Steps through one token and return the token and the type
}

func ParseFilterExpression(expression string) (Expression, error) {
	_, _, err := NewFilterExpressionParser(expression)
	if err != nil {
		return emptyExpression, err
	}

	//	converter := &FEConverter{expression: fe}
	//	converter.Init()
	//	converter.ConvertToAST()

	return emptyExpression, err
}
