// Copyright 2018-2019 Couchbase, Inc. All rights reserved.

package gojsonsm

import (
	"fmt"
	"github.com/alecthomas/participle"
	"strings"
)

// EBNF Grammar describing the parser

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

type FilterExpression struct {
	AndConditions []*FEAndCondition `@@ { "OR" @@ }`
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
	var err error
	f.i = 0
	for i := 0; i < len(f.fe.AndConditions); i++ {
		f.fe.AndConditions[i].stHelper = &feAndConditionST{e: f.fe.AndConditions[i]}
		err = f.fe.AndConditions[i].stHelper.Init()
		if err != nil {
			return err
		}
	}
	return nil
}

// Need to write unit test
func (f *filterExpressionST) Done() bool {
	if f.i < len(f.fe.AndConditions)-1 {
		return false
	}

	return f.fe.AndConditions[f.i].stHelper.Done()
}

type FEParen struct {
	OpenParen  *FEOpenParen  `@@ |`
	CloseParen *FECloseParen `@@`
	stHelper   StepThroughIface
}

func (fep *FEParen) IsOpen() bool {
	return fep.OpenParen != nil && fep.CloseParen == nil
}

func (fep *FEParen) IsClose() bool {
	return fep.CloseParen != nil && fep.OpenParen == nil
}

func (fep *FEParen) String() string {
	if fep.IsOpen() {
		return fep.OpenParen.String()
	} else if fep.IsClose() {
		return fep.CloseParen.String()
	} else {
		return "?? (FEParen)"
	}
}

type FEOpenParen struct {
	Parens   string `@"("`
	stHelper StepThroughIface
}

func (feop *FEOpenParen) String() string {
	return "("
}

type FECloseParen struct {
	Parens   string `@")"`
	stHelper StepThroughIface
}

func (fecp *FECloseParen) String() string {
	return ")"
}

type FEAndCondition struct {
	OrConditions []*FECondition `@@ { "AND" @@ }`
	stHelper     StepThroughIface
}

func (ac *FEAndCondition) String() string {
	output := []string{}

	first := true
	for _, e := range ac.OrConditions {
		if first {
			first = false
		} else {
			output = append(output, "AND")
		}
		output = append(output, e.String())
	}

	return strings.Join(output, " ")
}

type feAndConditionST struct {
	e *FEAndCondition

	i int
}

func (e *feAndConditionST) IsTerm() bool {
	return false
}

func (e *feAndConditionST) Done() bool {
	// TODO
	return false
}

func (e *feAndConditionST) Init() error {
	return nil
}

type FECondition struct {
	PreParen  []*FEOpenParen  `{ @@ }`
	Not       *FECondition    `"NOT" @@`
	Operand   *FEOperand      `| @@`
	PostParen []*FECloseParen `{ @@ }`
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
	} else {
		outputStr = append(outputStr, "?? (FECondition)")
	}

	for i := 0; i < len(fec.PostParen); i++ {
		outputStr = append(outputStr, fec.PostParen[i].String())
	}

	return strings.Join(outputStr, " ")
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

type FEArrayIndex struct {
	ArrayIndex string `"[" [ @"-" ] @Int "]"`
	stHelper   StepThroughIface
}

func (i *FEArrayIndex) String() string {
	return fmt.Sprintf("[%v]", i.ArrayIndex)
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

// We have to do this funky way of matching because our FEOperand expression may not be composed of a compareOp
// And due to the complicated FEOperand op, we have to do char by char match so we can catch the not-matched case
// and go to the other type of operands

type FEOpChar struct {
	Equal       *bool `( @"=" |`
	LessThan    *bool `@"<" |`
	GreaterThan *bool `@">" )`
	stHelper    StepThroughIface
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

type FEConstFuncOneArg struct {
	ConstFuncOneArgName *FEConstFuncOneArgName `( @@ "("`
	Argument            *FEConstFuncArgument   `@@ ")" )`
	stHelper            StepThroughIface
}

func (oa *FEConstFuncOneArg) String() string {
	return fmt.Sprintf("%v( %v )", oa.ConstFuncOneArgName.String(), oa.Argument.String())
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

type FEConstFuncTwoArgs struct {
	ConstFuncTwoArgsName *FEConstFuncTwoArgsName `( @@ "("`
	Argument0            *FEConstFuncArgument    `@@ "," `
	Argument1            *FEConstFuncArgument    `@@ ")" )`
	stHelper             StepThroughIface
}

func (fta *FEConstFuncTwoArgs) String() string {
	return fmt.Sprintf("%v( %v , %v )", fta.ConstFuncTwoArgsName.String(), fta.Argument0.String(), fta.Argument1.String())
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
