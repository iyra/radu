package main

import "fmt"
import "strconv"
import "errors"
import "unicode"
import "strings"

//import "io"
//import "os"
//import "bytes"
import "bufio"
import "os"

const (
	t_symbol       = iota
	t_tree         = iota
	t_number_float = iota
	t_number_int   = iota
	t_number_rat   = iota
	t_head_symbol  = iota
	t_function     = iota
)

var typenames = map[int]string{
	t_symbol:       "symbol",
	t_tree:         "tree",
	t_number_float: "float",
	t_number_int:   "int",
	t_number_rat:   "rational",
	t_head_symbol:  "head-symbol",
	t_function:     "function",
}

type rational struct {
	num int64
	den int64
}

type number_value struct {
	floatval float64
	intval   int64
	ratval   rational
}

/* (lambda (x y) (+ x y))
args = ["x", "y"]
ast = (+ -> x -> y)

((lambda (x y) (+ x y)) 2 1)
eval(ast{(+ -> x -> y)} env{x: 2, y: 1}) => 3
*/

type function_value struct {
	args   [][]rune
	action *tree
}

type value struct {
	decorations []rune
	valtype     int
	symbol      []rune
	ast         *tree
	number      number_value
	function    function_value
}

type tree struct {
	val      value
	done_val bool
	next     *tree
	parent   *tree
}
type env struct {
	values map[string]value
	prev   *env
}

type convError struct {
	from string
	to   string
}

func (e *convError) Error() string {
	return fmt.Sprintf("convError: Can't convert %s to %s", e.from, e.to)
}

func show_value(valtype string, val string) string {
	if val != "" {
		return fmt.Sprintf("[%s %s]", valtype, val)
	} else {
		return fmt.Sprintf("[%s]", valtype)
	}
}

func value_symbol_init(name []rune) value {
	return value{make([]rune, 0), t_symbol, name, nil, number_value{0, 0, rational{0, 0}}, function_value{make([][]rune, 0), nil}}
}

func value_head_symbol_init(name []rune) value {
	return value{make([]rune, 0), t_head_symbol, name, nil, number_value{0, 0, rational{0, 0}}, function_value{make([][]rune, 0), nil}}
}

func value_ast_init(ast *tree) value {
	return value{make([]rune, 0), t_tree, make([]rune, 0), ast, number_value{0, 0, rational{0, 0}}, function_value{make([][]rune, 0), nil}}
}

func value_number_int_init(n int64) value {
	return value{make([]rune, 0), t_number_int, make([]rune, 0), nil, number_value{0, n, rational{0, 0}}, function_value{make([][]rune, 0), nil}}
}

func value_number_float_init(n float64) value {
	return value{make([]rune, 0), t_number_float, make([]rune, 0), nil, number_value{n, 0, rational{0, 0}}, function_value{make([][]rune, 0), nil}}
}

func value_function_init(args [][]rune, action *tree) value {
	return value{make([]rune, 0), t_function, make([]rune, 0), nil, number_value{0, 0, rational{0, 0}}, function_value{args, action}}
}

func parse(input []rune, n int, ast *tree, dec []rune) int {
	if n == len(input) {
		//fmt.Printf("done")
	} else {
		switch c := input[n]; c {
		case '(':
			if ast.done_val {
				// value has finished collecting
				// now collect arguments
				//tree { value { make([]rune, 0), nil }, false, nil};
				// just move on because we do the tree allocation and nexting with ' '
				parse(input, n+1, ast, dec)
			} else {
				if len(ast.val.symbol) == 0 {
					// case like ((... so parse
					ast.val.decorations = dec
					ast.val.valtype = t_tree
					ast.val.ast = &tree{value_symbol_init(make([]rune, 0)), false, nil, ast}
					parse(input, n+1, ast.val.ast, make([]rune, 0))
				} else {
					fmt.Printf("error: unexpected ( in tree value\n")
				}
			}
			break
		case ')':
			ast.done_val = true
			if ast.parent != nil {
				return parse(input, n+1, ast.parent, make([]rune, 0))
			}
			break
		case ' ', '\n':
			if !ast.done_val {
				ast.done_val = true
			}
			if n+1 != len(input) {
				// finish off the last item
				ast.next = &tree{value_symbol_init(make([]rune, 0)), false, nil, ast.parent}
				if input[n+1] != ' ' {
					// get next argument
					parse(input, n+1, ast.next, make([]rune, 0))
				} else {
					for g := n; g < len(input); g++ {
						if input[g] != ' ' {
							parse(input, g, ast.next, make([]rune, 0))
							break
						}
					}
				}
			}
			break
		default:
			if len(ast.val.symbol) == 0 && (input[n] == ',' || input[n] == '\'' || input[n] == '`' || input[n] == '@') {
				ast.val.decorations = append(ast.val.decorations, input[n])
				parse(input, n+1, ast, append(dec, input[n])) // set the next thing to be escaped, whatever it is
			} else {
				ast.val.symbol = append(ast.val.symbol, input[n])
				parse(input, n+1, ast, dec)
			}
		}
	}
	return 0
}

func print_tree(ast *tree) {
	if ast != nil {
		if ast.val.ast == nil {
			print_value(ast.val)
		} else {
			fmt.Printf("(")
			print_tree(ast.val.ast)
			fmt.Printf(")")
		}
		//fmt.Printf("[%s]", typenames[ast.val.valtype])
		if ast.next != nil {
			fmt.Printf(" -> ")
			print_tree(ast.next)
		}
	} else {
		fmt.Printf("()")
	}
}

/*
(+ 3 2)
((x) y)
(p z)
*/

func blank_value() value {
	return value{make([]rune, 0), t_symbol, make([]rune, 0), nil, number_value{0, 0, rational{0, 0}}, function_value{make([][]rune, 0), nil}}
}

func quotefunc(ast *tree, bindings *env) (value, error) {
	if ast.next != nil {
		if ast.next.val.valtype != t_tree {
			return value_ast_init(&tree{ast.next.val, false, nil, nil}), nil
		} else {
			return ast.next.val, nil
		}
	}
	return blank_value(), errors.New("usage: (quote <value>)")
}

func listeval(ast *tree, bindings *env, original *tree) (*tree, error) {
	var err error
	ast.val, err = eval2(ast, bindings)
	if err != nil {
		return nil, err
	}
	if ast.next != nil {
		return listeval(ast.next, bindings, original)
	}
	//fmt.Println("original: ")
	return original, nil
}

func listfunc(ast *tree, bindings *env, ret *tree) (*tree, error) {
	/* (x -> y -> z)[tree] */
	if ast.next == nil {
		//return nil, errors.New("usage: (list x[ y z]); members will be evaluated.")
		return &tree{value_ast_init(nil), true, nil, nil}, nil
	}
	if r, err := listeval(ast.next, bindings, ast.next); err == nil {
		//print_value(value{t_tree, make([]rune, 0), r, number_value{0, 0}, function_value{make([][]rune, 0), nil}})

		return &tree{value_ast_init(r), true, nil, nil}, nil
	} else {
		return nil, err
	}
}

func consfunc(ast *tree, bindings *env, orig *tree) (value, error) {
	return blank_value(), nil
}

func is_symbol(v value) bool {
	return (v.valtype == t_head_symbol || v.valtype == t_symbol)
}

func lambda_arglist(ast *tree, agg [][]rune, bindings *env) ([][]rune, error) {
	print_tree(ast)
	/*p, err := eval2(ast, bindings)
	if err != nil {
		return make([][]rune, 0), err
	}
	print_value(p)
	if !is_symbol(p) {
		//fmt.Println("prob")
		return make([][]rune, 0), errors.New(fmt.Sprintf("error: lambda arglist must contain symbols only, given %s", typenames[p.valtype]))
	}*/
	//fmt.Println("PUSHING ", string(p.symbol))
	if ast.next == nil {
		return append(agg, ast.val.symbol), nil
	} else {
		return lambda_arglist(ast.next, append(agg, ast.val.symbol), bindings)
	}
}

func lambdafunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil || ast.next.next == nil {
		return blank_value(), errors.New("usage: (lambda (arg1 arg arg3 ...) (body)")
	}
	if ast.next.val.valtype != t_tree && ast.next.val.ast.val.valtype != t_symbol {
		return blank_value(), errors.New(fmt.Sprintf("error: lambda arglist must be type_tree, given: %s", typenames[ast.next.val.valtype]))
	}
	if arglist, err := lambda_arglist(ast.next.val.ast, make([][]rune, 0), bindings); err == nil {
		/*for _, r := range arglist {
			fmt.Println("ss ", string(r))
		}*/
		return value_function_init(arglist, ast.next.next), nil
	} else {
		return blank_value(), err
	}

	return blank_value(), nil
}

func argcount(ast *tree, total int) int {
	if ast.next != nil {
		return argcount(ast, total+1)
	}
	return total
}

func nth_rune(str string, n int) (rune, error) {
	for i, v := range str {
		if i == n {
			return v, nil
		}
	}
	return rune('0'), errors.New("n out of range")
}

func is_integer(symbol []rune) bool {
	for i, e := range symbol {
		if !unicode.IsDigit(e) {
			// a - at the start is OK because negative numbers are OK
			if !(i == 0 && symbol[0] == '-') {
				return false
			}
		}
	}
	return true
}

func conv_integer(symbol []rune) (int64, error) {
	//fmt.Println("converting ", string(symbol))
	return strconv.ParseInt(string(symbol), 10, 64)
}

func is_float(symbol []rune) bool {
	if strings.Count(string(symbol), ".") == 1 {
		//fmt.Printf("there is only one . in %s", string(symbol))
		for _, e := range symbol {
			if !unicode.IsDigit(e) && e != '.' {
				return false
			}
		}
		return true
	}
	return false
}

func conv_float(symbol []rune) (float64, error) {
	return strconv.ParseFloat(string(symbol), 64)
}

func bound(symbol []rune, bindings *env) (value, error) {
	/*for k, u := range bindings.values {
		fmt.Printf("%s: ", k)
		print_value(u)
		fmt.Println()
	}*/
	if val, ok := bindings.values[string(symbol)]; ok {
		return val, nil
	}
	if bindings.prev != nil {
		fmt.Println("looking in prev")
		return bound(symbol, bindings.prev)
	} else {
		return blank_value(),
			errors.New(fmt.Sprintf("error: symbol %s not found in environment.", string(symbol)))
	}
}

func collect_number_values(ast *tree,
	bindings *env,
	vlist []value) ([]value, error) {
	if ast == nil {
		return vlist, nil
	}
	if g, err := eval2(ast, bindings); err == nil {
		if g.valtype == t_number_int || g.valtype == t_number_float || g.valtype == t_number_rat {
			return collect_number_values(ast.next, bindings, append(vlist, g))
		} else {
			return make([]value, 0), errors.New(fmt.Sprintf("error: expected number, got %s", typenames[g.valtype]))
		}
	} else {
		return make([]value, 0), err
	}
}

func number_result(nlist []value) int {
	float_count, int_count, rational_count := 0, 0, 0
	for _, e := range nlist {
		if e.valtype == t_number_float {
			float_count += 1
		}
		if e.valtype == t_number_int {
			int_count += 1
		}
		if e.valtype == t_number_rat {
			rational_count += 1
		}
	}
	if float_count > 0 {
		return t_number_float
	}
	if int_count == len(nlist) {
		return t_number_int
	}
	// only remaining possibility is combination of ints and rationals
	return t_number_rat
}

func num2rat(v value) rational {
	switch v.valtype {
	case t_number_rat:
		return v.number.ratval
	case t_number_int:
		return rational{v.number.intval, 1}
	case t_number_float:
		/* return this because we have no other choice */
		return rational{int64(v.number.floatval), 1}
	}
	return rational{0, 0}
}

func num2int(v value) int64 {
	switch v.valtype {
	case t_number_int:
		return v.number.intval
	case t_number_float:
		return int64(v.number.floatval)
	case t_number_rat:
		return int64(num2float(v))
	}
	return int64(0)
}

func num2float(v value) float64 {
	switch v.valtype {
	case t_number_int:
		return float64(v.number.intval)
	case t_number_float:
		return v.number.floatval
	case t_number_rat:
		return float64(v.number.ratval.num / v.number.ratval.den)
	}
	return float64(0)
}

func subfunc(ast *tree, bindings *env) (value, error) {
	vlist, err := collect_number_values(ast.next, bindings, make([]value, 0))
	if err == nil {
		u := number_result(vlist)
		switch u {
		case t_number_float:
			t := num2float(vlist[0])
			for _, v := range vlist[1:] {
				t -= num2float(v)
			}
			return value_number_float_init(t), nil
		case t_number_int:
			t := num2int(vlist[0])
			for _, v := range vlist[1:] {
				t -= num2int(v)
			}
			return value_number_int_init(t), nil
		}
		return blank_value(), errors.New(fmt.Sprintf("error: couldn't match %s as number type", typenames[u]))
	} else {
		return blank_value(), err
	}
}

func addfunc(ast *tree, bindings *env) (value, error) {
	vlist, err := collect_number_values(ast.next, bindings, make([]value, 0))
	if err == nil {
		u := number_result(vlist)
		switch u {
		case t_number_float:
			t := float64(0)
			for _, v := range vlist {
				t += num2float(v)
			}
			return value_number_float_init(t), nil
		case t_number_int:
			t := int64(0)
			for _, v := range vlist {
				t += num2int(v)
			}
			return value_number_int_init(t), nil
		}
		return blank_value(), errors.New(fmt.Sprintf("error: couldn't match %s as number type", typenames[u]))
	} else {
		return blank_value(), err
	}
}

func multfunc(ast *tree, bindings *env) (value, error) {
	vlist, err := collect_number_values(ast.next, bindings, make([]value, 0))
	if err == nil {
		u := number_result(vlist)
		switch u {
		case t_number_float:
			t := float64(1)
			for _, v := range vlist {
				t *= num2float(v)
			}
			return value_number_float_init(t), nil
		case t_number_int:
			t := int64(1)
			for _, v := range vlist {
				t *= num2int(v)
			}
			return value_number_int_init(t), nil
		}
		return blank_value(), errors.New(fmt.Sprintf("error: couldn't match %s as number type", typenames[u]))
	} else {
		return blank_value(), err
	}
}

/*func divfunc(ast *tree, bindings *env) (value, error) {
	vlist, err := collect_number_values(ast.next, bindings, make([]value, 0))
	if err == nil {
		u := number_result(vlist)
		switch u {
		case t_number_float:
			t := float64(vlist[0])
			for _, v := range vlist[1:] {
				t *= num2float(v)
			}
			return value_number_float_init(t), nil
		case t_number_int:
			t := int64(vlist[0])
			for _, v := range vlist[1:] {
				t *= num2int(v)
			}
			return value_number_int_init(t), nil
		}
		return blank_value(), errors.New(fmt.Sprintf("error: couldn't match %s as number type", typenames[u]))
	} else {
		return blank_value(), err
	}
}*/

func succfunc(ast *tree, bindings *env) (value, error) {
	if item, err := eval2(ast.next, bindings); err == nil {
		if item.valtype == t_number_int {
			return value_number_int_init(item.number.intval + 1), nil
		}
		return blank_value(), errors.New(fmt.Sprintf("wrong type %s to succ; number_int expected.", typenames[item.valtype]))
	} else {
		return blank_value(), err
	}
}

func get_subjects(subject *tree, results []value, bindings *env) ([]value, error) {
	if g, err := eval2(subject, bindings); err == nil {
		results = append(results, g)
	} else {
		return make([]value, 0), err
	}
	if subject.next != nil {
		return get_subjects(subject.next, results, bindings)
	}
	return results, nil
}

func performfunc(v value, bindings *env, subject *tree) (value, error) {
	//print_value(v)

	/* set the bindings inside our lambda to be the same as the outside ones
	but overwrite the ones named in the varlist */
	//local_bindings := bindings.values
	if g, err := get_subjects(subject, make([]value, 0), bindings); err == nil && len(g) == len(v.function.args) {
		for i, e := range v.function.args {
			bindings.values[string(e)] = g[i]
		}
	} else {
		if err != nil {
			return blank_value(), err
		}
		//fmt.Printf("len get_subjects = %d, len func.args = %d\n", len(g), len(v.function.args))
		return blank_value(), errors.New("error: mismatched arg length for lambda")
	}
	return eval2(v.function.action, bindings)
}

/*func eval(ast *tree, bindings *env) (value, error) {
	return blank_value(), nil
}*/

func topeval(ast *tree, bindings *env) value {
	/* use this function to cycle through ast.next at the top level
	and evaluate each in turn, returning the last value
	problem is that we need the bindings from each previous one
	so make eval2 return bindings? */
	return blank_value()
}

func evalfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil {
		return blank_value(), errors.New("usage: (eval x)")
	}
	g, err := eval2(ast.next, bindings)
	if err == nil {
		if g.valtype == t_tree {
			return eval2(g.ast, bindings)
		}
		return g, nil
	} else {
		return blank_value(), err
	}
}

func carfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil {
		return blank_value(), errors.New("usage: (car (list x[ y z ...]))")
	}
	/*if ast.next.val.valtype != t_tree {
		return blank_value(), errors.New("error: car only accepts list")
	}*/
	if v, e := eval2(ast.next, bindings); e == nil && v.valtype == t_tree {
		if v.ast.val.ast != nil {
			return v.ast.val.ast.val, nil
		}
		return v.ast.val, nil
	} else {
		return blank_value(), e
	}
	//return blank_value(), nil
}

func cdrfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil {
		return blank_value(), errors.New("usage: (cdr (list x[ y z ...]))")
	}
	if v, e := eval2(ast.next, bindings); e == nil && v.valtype == t_tree {
		//fmt.Println("ru")
		print_value(v)
		if v.ast == nil {
			return blank_value(), errors.New("error: can't cdr an empty list")
		}
		if v.ast.val.ast.next == nil {
			return value_ast_init(nil), nil
			//return blank_value(), errors.New("error: can only cdr a list with more than one value")
		}
		print_value(value_ast_init(&tree{value_ast_init(v.ast.val.ast.next), true, nil, nil}))
		return value_ast_init(&tree{value_ast_init(v.ast.val.ast.next), true, nil, nil}), nil
	} else {
		return blank_value(), e
	}
}

func cadrfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil {
		return blank_value(), errors.New("usage: (cadr (list x[ y z ...]))")
	}
	if v, e := cdrfunc(ast, bindings); e == nil {
		//fmt.Println("ut")
		print_tree(&tree{blank_value(), true, v.ast, nil})
		return carfunc(&tree{blank_value(), true, v.ast, nil}, bindings)
	} else {
		return blank_value(), e
	}
}

func let_binds(b *tree, names [][]rune, values []value, bindings *env) ([][]rune, []value, error) {
	//fmt.Println("from let_binds: ")
	if b == nil {
		// for _, n := range names {
		// 	fmt.Println(string(n))
		// }
		// for _, v := range values {
		// 	print_value(v)
		// 	fmt.Println()
		// }
		return names, values, nil
	}
	if b.val.valtype == t_tree {
		//print_tree(b.val.ast)
		//print_value(b.ast)
		if b.val.ast.val.valtype == t_symbol || b.val.ast.val.valtype == t_head_symbol {
			if b.val.ast.next != nil {
				if r, e := eval2(b.val.ast.next, bindings); e == nil {
					return let_binds(b.next, append(names, b.val.ast.val.symbol), append(values, r), bindings)
				} else {
					return nil, nil, e
				}
			} else {
				return nil, nil, errors.New(fmt.Sprintf("error: let binding must have value component; symbol: %s", b.val.ast.val.symbol))
			}
		} else {
			return nil, nil, errors.New(fmt.Sprintf("error: let binding must bind to symbol, given type: %s", typenames[b.val.ast.val.valtype]))
		}
		return make([][]rune, 0), make([]value, 0), nil
	} else {
		return nil, nil, errors.New("error: expected a tree in let bind")
	}
}

func bind_let(kvs *tree, bindings *env) (*env, error) {
	names, values, err := let_binds(kvs.val.ast, nil, nil, bindings)
	if err == nil {
		for i, v := range names {
			bindings.values[string(v)] = values[i]
		}
		return bindings, nil
	} else {
		return nil, err
	}
}

func letfunc(ast *tree, bindings *env) (value, error) {
	/* (let ((x 1) (b 2)) (+ x b)) */
	if ast.next == nil || ast.next.next == nil {
		return blank_value(), errors.New("usage: (let ((var1 val1)[ (val2 val2) ...]) function)")
	}
	//fmt.Println("binds:")
	print_tree(ast.next)
	//fmt.Println("function:")
	print_tree(ast.next.next)
	//fmt.Println("bind_let:")
	if v, e := bind_let(ast.next, bindings); e == nil {
		//fmt.Println("letfunc evaling")
		//fmt.Println("ast.next.next is ")
		print_tree(ast.next.next)
		g, e2 := eval2(ast.next.next, v)
		bindings = bindings.prev
		return g, e2
	} else {
		return blank_value(), e
	}
}

func prognfunc(ast *tree, bindings *env) (value, error) {
	r, e := eval2(ast, bindings)
	if e == nil {

	} else {
		return blank_value(), e
	}
	if ast.next != nil {
		//fmt.Println("there is next")
		prognfunc(ast.next, bindings)
	} else {
		return r, nil
	}
	return blank_value(), nil
}

func truesym() value {
	return value_symbol_init([]rune("#t"))
}

func falsesym() value {
	return value_symbol_init([]rune("#f"))
}

func equaltrees(ast1 *tree, ast2 *tree, bindings *env) (bool, error) {
	if ast1 == nil && ast2 == nil {
		return true, nil
	}
	if ast1 == nil && ast2 != nil {
		return false, nil
	}
	if ast1 != nil && ast2 == nil {
		return false, nil
	}
	if ast1.next == nil && ast2.next == nil {
		if v1, e1 := eval2(ast1, bindings); e1 == nil {
			if v2, e2 := eval2(ast2, bindings); e2 == nil {
				return equalvals(v1, v2, bindings)
			} else {
				return false, e2
			}
		} else {
			return false, e1
		}
	}
	if ast1.next != nil && ast2.next != nil {
		if v1, e1 := eval2(ast1.next, bindings); e1 == nil {
			if v2, e2 := eval2(ast2.next, bindings); e2 == nil {
				if g, e3 := equalvals(v1, v2, bindings); e3 == nil && g {
					return equaltrees(ast1.next, ast2.next, bindings)
				} else {
					return false, e3
				}
			} else {
				return false, e2
			}
		} else {
			return false, e1
		}
	}
	return false, nil
}

func equalvals(v1 value, v2 value, bindings *env) (bool, error) {
	if v1.valtype == v2.valtype {
		switch v1.valtype {
		case t_number_int:
			if v1.number.intval == v2.number.intval {
				return true, nil
			}
		case t_number_float:
			//fmt.Println(v1.valtype, v2.valtype)
			if v1.number.floatval == v2.number.floatval {
				return true, nil
			}
		case t_symbol, t_head_symbol:
			for i, r := range v1.symbol {
				if r != v2.symbol[i] {
					return false, nil
				}
			}
			return true, nil
		case t_tree:
			return equaltrees(v1.ast, v2.ast, bindings)
		}
	} else {
		return false, errors.New(fmt.Sprintf("error: different types do not equal; given: %s, %s", typenames[v1.valtype], typenames[v2.valtype]))
	}
	return false, nil
}

func eqfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil {
		return blank_value(), errors.New("usage: (eq val1 val2)")
	}
	if v1, e1 := eval2(ast.next, bindings); e1 == nil {
		if v2, e2 := eval2(ast.next.next, bindings); e2 == nil {
			if g, e3 := equalvals(v1, v2, bindings); e3 == nil && g {
				return truesym(), nil
			} else {
				//fmt.Println("fff")
				return falsesym(), e3
			}
		} else {
			return blank_value(), e2
		}
	} else {
		return blank_value(), e1
	}
}

func isfalse(v value, bindings *env) (bool, error) {
	if v.valtype == t_symbol {
		if g, e := equalvals(v, falsesym(), bindings); e == nil {
			if g {
				return true, nil
			}
			return false, nil
		} else {
			return false, e
		}
	}
	return false, nil
}

func istrue(v value, bindings *env) (bool, error) {
	if p, e := isfalse(v, bindings); e == nil {
		if !p {
			//print_value(v)
			//fmt.Println("is true!")
			return true, nil
		}
		//fmt.Println("fug")
		return false, nil
	} else {
		return false, e
	}
}

func modfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil || ast.next.next == nil {
		return blank_value(), errors.New("usage: (% x y)")
	}
	if n1, e1 := eval2(ast.next, bindings); e1 == nil {
		if n1.valtype == t_number_int {
			if n2, e2 := eval2(ast.next.next, bindings); e2 == nil {
				if n2.valtype == t_number_int {
					if n2.number.intval == 0 {
						return blank_value(), errors.New("error: second argument to % cannot be 0")
					}
					return value_number_int_init(n1.number.intval % n2.number.intval), nil
				} else {
					return blank_value(), errors.New("error: arguments to mod must be integers")
				}
			} else {
				return blank_value(), e2
			}
		} else {
			return blank_value(), errors.New("error: arguments to mod must be integers")
		}
	} else {
		return blank_value(), e1
	}
}

func iffunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil || ast.next.next == nil || ast.next.next.next == nil {
		return blank_value(), errors.New("usage: (if condition evaluate-if-true evaluate-if-false)")
	}
	if v, e := eval2(ast.next, bindings); e == nil {
		if u, e2 := istrue(v, bindings); e2 == nil {
			if u {
				return eval2(ast.next.next, bindings)
			}
		} else {
			return blank_value(), e2
		}
	} else {
		return blank_value(), e
	}
	return eval2(ast.next.next.next, bindings)
}

func applyfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil || ast.next.next == nil {
		return blank_value(), errors.New("usage: (apply fn (list arg1[ arg2 arg3 ...]))")
	}
	if l, e := eval2(ast.next.next, bindings); e == nil {
		if l.ast != nil {
			return eval2(&tree{value_ast_init(&tree{ast.next.val, true, l.ast.val.ast, nil}), true, nil, nil}, bindings)
		} else {
			return blank_value(), errors.New("error: second argument to apply must be list")
		}
	} else {
		return blank_value(), e
	}
	return blank_value(), nil
}

func listdepth(ast *tree, i int64) int64 {
	if ast == nil {
		return 0
	}
	if ast.next == nil {
		return i
	}
	return listdepth(ast.next, i+1)
}

func lenfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil {
		return blank_value(), errors.New("usage: (len (list x[ y z ...]))")
	}
	if v, e := eval2(ast.next, bindings); e == nil {
		if v.valtype == t_tree {
			return value_number_int_init(listdepth(v.ast.val.ast, 1)), nil
		} else {
			return blank_value(), errors.New("error: lenfunc must be called on a list")
		}
	} else {
		return blank_value(), e
	}
}

func lastinlist(ast *tree) *tree {
	if ast == nil {
		// empty list
		return ast
	}
	if ast.next == nil {
		return ast
	}
	return lastinlist(ast.next)
}

func appendfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil || ast.next.next == nil {
		return blank_value(), errors.New("usage: (append item (list x[ y z ...]))")
	}
	if av, e0 := eval2(ast.next, bindings); e0 == nil {
		if v, e := eval2(ast.next.next, bindings); e == nil {
			if v.valtype == t_tree {
				lastptr := lastinlist(v.ast.val.ast)
				fmt.Println("lastptr is")
				print_tree(lastptr)
				if lastptr != nil {
					lastptr.next = &tree{av, true, nil, nil}
				} else {
					return value_ast_init(&tree{value_ast_init(&tree{av, true, nil, nil}), true, nil, nil}), nil
				}
				return v, nil
			} else {
				return blank_value(), errors.New("error: second argument to append must be list")
			}
		} else {
			return blank_value(), e
		}
	} else {
		return blank_value(), e0
	}
}

func prependfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil || ast.next.next == nil {
		return blank_value(), errors.New("usage: (prepend item (list x[ y z ...]))")
	}
	if av, e0 := eval2(ast.next, bindings); e0 == nil {
		if v, e := eval2(ast.next.next, bindings); e == nil {
			if v.valtype == t_tree {
				return value_ast_init(&tree{value_ast_init(&tree{av, true, v.ast.val.ast, nil}), true, nil, nil}), nil
			} else {
				return blank_value(), errors.New("error: second argument to append must be list")
			}
		} else {
			return blank_value(), e
		}
	} else {
		return blank_value(), e0
	}
}

func symisstring(sym []rune) bool {
	if sym[0] == '"' && sym[len(sym)-1] == '"' {
		return true
	}
	return false
}

func stringify(sym []rune) []rune {
	return sym[1 : len(sym)-1]
}

func strindexfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil {
		return blank_value(), errors.New("usage: (strindex \"my string\" n)")
	}
	if v, e := eval2(ast.next, bindings); e == nil {
		if v.valtype == t_symbol || v.valtype == t_head_symbol {
			if symisstring(v.symbol) {
				if posv, e2 := eval2(ast.next.next, bindings); e2 == nil {
					if posv.valtype == t_number_int {
						if posv.number.intval < 0 || posv.number.intval > int64(len(stringify(v.symbol))-1) {
							return blank_value(), errors.New(fmt.Sprintf("error: index %d for string %s out of range", posv.number.intval, string(v.symbol)))
						}
						g := make([]rune, 0)
						g = append(g, stringify(v.symbol)[posv.number.intval])
						return value_symbol_init(g), nil
					} else {
						return blank_value(), errors.New("error: second argument to strindex must be int")
					}
				} else {
					return blank_value(), e2
				}
			} else {
				return blank_value(), errors.New("error: first argument to strlen must be a symbol starting and ending with double quotes")
			}
		} else {
			return blank_value(), errors.New("error: first argument to strlen must be a symbol")
		}
	} else {
		return blank_value(), e
	}
}

func strlenfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil {
		return blank_value(), errors.New("usage: (strlen \"my string\")")
	}
	if v, e := eval2(ast.next, bindings); e == nil {
		if v.valtype == t_symbol || v.valtype == t_head_symbol {
			if symisstring(v.symbol) {
				return value_number_int_init(int64(len(stringify(v.symbol)))), nil
			} else {
				return blank_value(), errors.New("error: first argument to strlen must be a symbol starting and ending with double quotes")
			}
		} else {
			return blank_value(), errors.New("error: first argument to strlen must be a symbol")
		}
	} else {
		return blank_value(), e
	}
}

/* cat -> tree lol */
func cat(ast *tree, bindings *env, ret []rune) ([]rune, error) {
	if ast.next == nil {
		//fmt.Println("length of ret is ", len(ret))
		u := make([]rune, 1)
		u[0] = '"'
		for _, x := range ret {
			u = append(u, x)
		}
		u = append(u, '"')
		return u, nil
	}
	if v, e := eval2(ast.next, bindings); e == nil {
		if v.valtype == t_symbol || v.valtype == t_head_symbol {
			if symisstring(v.symbol) {
				//g := ret+v.symbol
				//ret = make([]rune, len(v.symbol)+len(ret))
				for _, r := range stringify(v.symbol) {
					ret = append(ret, r)
				}
				return cat(ast.next, bindings, ret)
			} else {
				return ret, errors.New("error: first argument to strlen must be a symbol starting and ending with double quotes")
			}
		} else {
			return ret, errors.New("error: first argument to strlen must be a symbol")
		}
	} else {
		return ret, e
	}
}

func strcatfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil || ast.next.next == nil {
		return blank_value(), errors.New("usage: (strcat \"str1\" \"str2\"[ \"str3\" ...])")
	}
	if result, e := cat(ast, bindings, make([]rune, 0)); e == nil {
		//fmt.Println(string(result))
		return value_symbol_init(result), nil
	} else {
		return blank_value(), e
	}
	return blank_value(), nil
}

func intfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil {
		return blank_value(), errors.New("usage: (int value)")
	}
	if v, e := eval2(ast.next, bindings); e == nil {
		switch v.valtype {
		case t_symbol:
			if symisstring(v.symbol) {
				if len(stringify(v.symbol)) == 1 {
					return value_number_int_init(int64(stringify(v.symbol)[0])), nil
				}
			}
		}
	}
	return blank_value(), nil
}

func collect_bools(ast *tree, bindings *env, ret []bool) ([]bool, error) {
	if b, e := istrue(ast.val, bindings); e == nil {
		if ast.next != nil {
			return collect_bools(ast.next, bindings, append(ret, b))
		} else {
			return append(ret, b), nil
		}
	} else {
		return nil, e
	}
}

func nandfunc(ast *tree, bindings *env) (value, error) {
	if ast.next == nil || ast.next.next == nil {
		return blank_value(), errors.New("usage: (nand bool1 bool2[ bool3 bool4 ...])")
	}
	if bs, e := collect_bools(ast.next, bindings, make([]bool, 0)); e == nil {
		for _, b := range bs {
			if !b {
				return truesym(), nil
			}
		}
		return falsesym(), nil
	} else {
		return falsesym(), e
	}
}

func vals2list(vs []value, i int, target *tree) {
	target.val = vs[i]
	if i != len(vs)-1 {
		target.next = &tree{blank_value(), true, nil, nil}
		vals2list(vs, i+1, target.next)
	}
}

func applyeach(fn value, l *tree, b *env, acc []value) ([]value, error) {
	print_value(l.val)
	print_tree(&tree{value_ast_init(&tree{fn, true, &tree{l.val, true, nil, nil}, nil}), true, nil, nil})
	if r, e := eval2(&tree{value_ast_init(&tree{fn, true, &tree{l.val, true, nil, nil}, nil}), true, nil, nil}, b); e == nil {
		//if r, e := eval2(&tree{fn, true, &tree{l.val, true, nil, nil}, nil}, b); e == nil {
		print_value(r)
		acc = append(acc, r)
	} else {
		return make([]value, 0), e
	}
	if l.next != nil {
		return applyeach(fn, l.next, b, acc)
	} else {
		return acc, nil
	}
}

func dofor(ast *tree, bindings *env) (value, error) {
	if ast.next != nil && ast.next.next != nil {
		if fn, ev := eval2(ast.next, bindings); ev == nil {
			print_value(fn)
			if v, e0 := eval2(ast.next.next, bindings); e0 == nil {
				if v.valtype == t_tree {
					vals, e1 := applyeach(fn, v.ast.val.ast, bindings, make([]value, 0))
					if e1 == nil {
						x := tree{blank_value(), true, nil, nil}
						vals2list(vals, 0, &x)
						return value_ast_init(&tree{value_ast_init(&x), true, nil, nil}), nil
					} else {
						return blank_value(), e1
					}
				} else {
					return blank_value(), errors.New("error: second argument to dofor must be list")
				}
			} else {
				return blank_value(), e0
			}
		} else {
			return blank_value(), ev
		}
	} else {
		return blank_value(), errors.New("usage: (dofor fn (list x[ y z ...]))")
	}
}

func definefunc(ast *tree, bindings *env) (value, error) {
	if ast.next != nil && ast.next.next != nil {
		if ast.next.val.valtype != t_symbol {
			return blank_value(), errors.New(fmt.Sprintf("error: define can't bind to a non-symbol (%s)", typenames[ast.next.val.valtype]))
		}
		if g, e0 := eval2(ast.next.next, bindings); e0 == nil {
			bindings.prev.values[string(ast.next.val.symbol)] = g
			return blank_value(), nil
		} else {
			return blank_value(), e0
		}
	} else {
		return blank_value(), errors.New("usage: (define my-symbol value)")
	}
}

func funcdex(symbol []rune, ast *tree, bindings *env) (value, error) {
	switch sym := string(symbol); sym {
	case "quit", "exit":
		os.Exit(0)
	case "define":
		return definefunc(ast, &env{make(map[string]value), bindings})
	case "quote":
		//fmt.Println("doing quote")
		return quotefunc(ast, bindings)
	case "cons":
		return consfunc(ast, bindings, ast)
	case "list":
		if p, e := listfunc(ast, bindings, ast); e == nil {
			return value_ast_init(p), nil
		} else {
			return blank_value(), e
		}
	case "dofor":
		return dofor(ast, bindings)
	case "succ":
		fmt.Println(bindings)
		return succfunc(ast, bindings)
	case "+":
		return addfunc(ast, bindings)
	case "-":
		return subfunc(ast, bindings)
	case "*":
		return multfunc(ast, bindings)
	case "lambda":
		return lambdafunc(ast, bindings)
	case "eval":
		return evalfunc(ast, bindings)
	case "car":
		return carfunc(ast, bindings)
	case "cdr":
		return cdrfunc(ast, bindings)
	case "cadr":
		return cadrfunc(ast, bindings)
	case "let":
		v, e := letfunc(ast, &env{make(map[string]value), bindings})
		return v, e
	case "progn":
		/* progn has to be like this because we call it instead of eval
		after parsing the main tree */
		if ast.next != nil {
			return prognfunc(ast.next, bindings)
		} else {
			return blank_value(), errors.New("error: progn expects at least one argument")
		}
	case "eq":
		return eqfunc(ast, bindings)
	case "if":
		return iffunc(ast, bindings)
	case "%":
		return modfunc(ast, bindings)
	case "apply":
		return applyfunc(ast, bindings)
	case "len":
		return lenfunc(ast, bindings)
	case "append":
		return appendfunc(ast, bindings)
	case "prepend":
		return prependfunc(ast, bindings)
	case "strlen":
		return strlenfunc(ast, bindings)
	case "strindex":
		return strindexfunc(ast, bindings)
	case "strcat":
		return strcatfunc(ast, bindings)
	case "int":
		return intfunc(ast, bindings)
	case "nand":
		return nandfunc(ast, bindings)
	default:
		fmt.Println("GUNVGUN")
		fmt.Println("looking for ", string(ast.val.symbol))
		if res, finderr := bound(ast.val.symbol, bindings); finderr == nil {
			print_value(res)
			print_tree(&tree{value_ast_init(&tree{res, true, ast.next, nil}), true, nil, nil})
			return eval2(&tree{value_ast_init(&tree{res, true, ast.next, nil}), true, nil, nil}, bindings)
		} else {
			return blank_value(), finderr // couldn't find the x in (x y)
		}
	}
	return blank_value(), errors.New("fuck")
}

/* perhaps eval can be improved to evaluate a sequence of trees
like (x 1) (b 2) because at the moment it only evaluates (x 1) and then stops.

try to evaluate each one, but rather than returning the result, we see if there is
another tree following the thing we just evaluated
if there is not then we just return it; if there is then we eval the next tree

e.g (define x 3)
define is evaluated, ast.next is checked but it's nil, so just return the result of define
e.g (define x 3) (print x)
define is evaluated, ast.next is seen to contain another tree, so evaluate that
*/

func quoted_sym(sym []rune) (bool, []rune) {
	if sym[0] == '\'' {
		return true, sym[1:]
	} else {
		return false, sym
	}
}

func eval2(ast *tree, bindings *env) (value, error) {
	/*
						1. ast = (fn arg1 arg2 arg3 ...) => call fnfunc() with ast
							2. ((fn arg1 arg2 arg3 ...) ext1 ext2 ext3) => evaluate ast.next then if ast.next is a lambda then performfunc, else fnfunc() lookup
							3. 'sym => produce sym
							4. 3 => produce int 3
							5. 3.0 => produce int 3.0
							6. 3/5 => produce rational 3/5
							7. '3 => produce int 3
							8. '3.0 => produce float 3.0
							9. '3/5 => produce rational 3/5
							10. sym => if sym fits pattern of a number then return number value, otherwise lookup sym and return value
							11. '(arg1 arg2 arg3) => treat like (list 'arg1 'arg2 'arg3)
				12. - => return symbol -
			13. fn => return fn
		14. #t => #t
		15. #f => #f */

	if ast.val.valtype == t_tree {
		if ast.val.ast != nil {
			if len(ast.val.decorations) == 0 {
				/* no quotes or anything, so just evaluate */
				// case 1
				if ast.val.ast.val.valtype == t_symbol {
					return funcdex(ast.val.ast.val.symbol, ast.val.ast, bindings)
				}

				// case 2
				if ast.val.ast.val.valtype == t_tree {
					print_tree(ast.val.ast)
					if af, e := eval2(ast.val.ast, bindings); e == nil {
						fmt.Println("g")
						if af.valtype == t_function {
							// a lambda
							fmt.Println("lambdaing")
							y, u := performfunc(af, &env{make(map[string]value, 0), bindings}, ast.val.ast.next)
							return y, u
						}
						if af.valtype == t_symbol {
							// a symbol
							return funcdex(af.symbol, ast.val.ast, bindings)
						}
					} else {
						return blank_value(), e
					}
				}

				if ast.val.ast.val.valtype == t_function {
					fmt.Println("got a tree")
					y, u := performfunc(ast.val.ast.val, &env{make(map[string]value, 0), bindings}, ast.val.ast.next)
					return y, u
				}
			} else {
				// case 11
				if ast.val.decorations[0] == '\'' {
					ast.val.decorations = ast.val.decorations[1:]
					return quotefunc(&tree{value_symbol_init([]rune("quote")), true, ast.val.ast, nil}, bindings)
				}

			}
		}
	}

	// case 4 & 5 & 13
	if ast.val.valtype == t_number_int || ast.val.valtype == t_number_float || ast.val.valtype == t_function {
		return ast.val, nil
	}

	if ast.val.valtype == t_symbol {
		// case 3
		//print_value(ast.val)
		if len(ast.val.decorations) > 0 && ast.val.decorations[0] == '\'' {
			//fmt.Println("ff")
			ast.val.decorations = ast.val.decorations[1:]
			return quotefunc(&tree{value_symbol_init([]rune("quote")), true, &tree{ast.val, true, nil, nil}, nil}, bindings)
		}

		rsym := ast.val.symbol

		// case 12
		if len(rsym) == 1 && rsym[0] == '-' {
			return ast.val, nil
		}

		// case 4 & 7
		if is_integer(rsym) {
			if v, err := conv_integer(rsym); err == nil {
				//fmt.Println("returning int")
				return value_number_int_init(v), nil
			} else {
				return blank_value(), err
			}
		}

		// case 5 & 8
		if is_float(rsym) {
			if v, err := conv_float(rsym); err == nil {
				return value_number_float_init(v), nil
			} else {
				return blank_value(), err
			}
		}

		// case 6 & 9
		// tbd

		// case 10
		if res, finderr := bound(ast.val.symbol, bindings); finderr == nil {
			return res, nil
		} else {
			if p, e3 := equalvals(ast.val, truesym(), bindings); e3 == nil && p {
				return ast.val, nil
			}
			if p, e3 := equalvals(ast.val, falsesym(), bindings); e3 == nil && p {
				return ast.val, nil
			}
			return blank_value(), finderr
		}
	}
	return blank_value(), nil
}

// func eval2(ast *tree, bindings *env) (value, error) {
// 	/* (x[*] y[*] z[*] ...)[tree]
// 		x must either be (1) builtin [val symbol] or (2) eval to a [val function_value]

// 	s
// 	s must either be (1) arbitrary but not tree [val symbol] (2) eval to a number (3) be a number
// 	*/
// 	//print_tree(ast)
// 	if ast.val.valtype == t_tree {
// 		if ast.val.ast == nil {
// 			return value_ast_init(nil), nil
// 		}

// 		if ast.val.ast.val.valtype == t_symbol || ast.val.ast.val.valtype == t_head_symbol {
// 			//print_tree(ast.val.ast)
// 			return funcdex(ast.val.ast.val.symbol, ast.val.ast, bindings)
// 		}

// 		if g, err := eval2(ast.val.ast, bindings); err == nil {
// 			if g.valtype == t_function {
// 				fmt.Println("yes lambda!")
// 				return performfunc(g, bindings, ast.val.ast.next)
// 			} else {
// 				if g.valtype == t_symbol || g.valtype == t_head_symbol {
// 					return funcdex(g.symbol, ast.val.ast, bindings)
// 				}
// 			}
// 		}
// 		//return blank_value(), errors.New("usage: (x [a b c ...]) where x is a builtin or lambda")
// 		/* i really shouldn't be doing this, but i need it to test list equality :/ */
// 		return ast.val, nil
// 	}

// 	if ast.val.valtype == t_symbol || ast.val.valtype == t_head_symbol {
// 		rsym := ast.val.symbol
// 		// like the y or z in (x y z)
// 		// or any other symbol sent to evaluate
// 		//symbols evaluate to numbers if they are numbers
// 		//fmt.Println("rsym is ", string(ast.val.symbol))
// 		//fmt.Println(is_integer(rsym), is_float(rsym))
// 		if len(rsym) == 1 && rsym[0] == '-' {
// 			return ast.val, nil
// 		}
// 		if is_integer(rsym) {
// 			if v, err := conv_integer(rsym); err == nil {
// 				//fmt.Println("returning int")
// 				return value_number_int_init(v), nil
// 			} else {
// 				return blank_value(), err
// 			}
// 		}

// 		if is_float(rsym) {
// 			if v, err := conv_float(rsym); err == nil {
// 				return value_number_float_init(v), nil
// 			} else {
// 				return blank_value(), err
// 			}
// 		}

// 		if qs, v := quoted_sym(rsym); qs {
// 			return value_symbol_init(v), nil
// 		}

// 		if res, finderr := bound(ast.val.symbol, bindings); finderr == nil {
// 			return res, nil
// 		} else {
// 			/* returning an error here seems to break lambda... perhaps we shouldn't eval the lambda arglist contents */
// 			return ast.val, nil
// 		}
// 		fmt.Println("juts returning")
// 		// it's not a number so just return it
// 		return ast.val, nil
// 	}

// 	if ast.val.valtype == t_number_int ||
// 		ast.val.valtype == t_number_float || ast.val.valtype == t_function {
// 		return ast.val, nil
// 	}
// 	if ast.next != nil {
// 		return eval2(ast.next, bindings)
// 	}
// 	return blank_value(), errors.New("nothing to do")
// }

/*func eval(ast *tree, bindings *env) (value, error) {
	fmt.Println("valtype is ", typenames[ast.val.valtype])
	if ast.parent == nil {
		fmt.Printf("parent is nil")
	}
	switch ast.val.valtype {
	case t_function:
		fmt.Println("got function")
	case t_head_symbol:
		// like (x y z), we are talking about x
		// apply to args
		switch sym := string(ast.val.symbol); sym {
		case "quote":
			fmt.Println("doing quote")
			return quotefunc(ast, bindings)
		case "cons":
			return consfunc(ast, bindings, ast)
		case "list":
			return listfunc(ast, bindings, ast)
		case "succ":
			return succfunc(ast, bindings)
		case "+":
			return addfunc(ast, bindings)
		case "lambda":
			return lambdafunc(ast, bindings)
		default:
			if res, finderr := bound(ast.val.symbol, bindings); finderr == nil {
				return res, nil
			} else {
				return blank_value(), finderr // couldn't find the x in (x y)
			}
		}
	case t_symbol:
		rsym := ast.val.symbol
		// like the y or z in (x y z)
		// or any other symbol sent to evaluate
		//symbols evaluate to numbers if they are numbers
		fmt.Println("rsym is ", ast.val.symbol)
		fmt.Println(is_integer(rsym), is_float(rsym))
		if is_integer(rsym) {
			if v, err := conv_integer(rsym); err == nil {
				return value_number_int_init(v), nil
			} else {
				return blank_value(), err
			}
		}
		if is_float(rsym) {
			if v, err := conv_float(rsym); err == nil {
				return value_number_float_init(v), nil
			} else {
				return blank_value(), err
			}
		}

		// it's not a number so just return it
		return ast.val, nil
	case t_number_float, t_number_int, t_number_rational:
		return ast.val, nil
	case t_tree:
		return eval(ast.val.ast, bindings)
	}
	if ast.parent == nil && ast.next != nil {
		fmt.Println("no parent")
		eval(ast.next, bindings)
	}
	//return eval(ast.val.ast, bindings)
	fmt.Println("hi")
	return eval(ast.next, bindings)
}*/

func print_value(v value) {
	fmt.Printf("%s", string(v.decorations))
	switch v.valtype {
	case t_symbol, t_head_symbol:
		fmt.Printf(string(v.symbol))
	case t_tree:
		print_tree(v.ast)
	case t_number_float:
		fmt.Printf("%f", v.number.floatval)
	case t_number_int:
		fmt.Printf("%d", v.number.intval)
	case t_function:
		fmt.Printf("inputs: ")
		for _, x := range v.function.args {
			fmt.Printf(string(x) + ", ")
		}
		fmt.Printf("action: ")
		print_tree(v.function.action)
	}
}

func repl(b *env) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("radu> ")
	text, _ := reader.ReadString('\n')
	my_tree := tree{value_symbol_init(make([]rune, 0)), false, nil, nil}
	program := text[:len(text)-1] // trim off the last character because it's a \n
	parse([]rune(program), 0, &my_tree, make([]rune, 0))
	r, err := prognfunc(&my_tree, b)
	if err == nil {
		print_value(r)
	} else {
		fmt.Printf(err.Error())
	}

	//print_tree(&my_tree)
	fmt.Println()
	repl(b)
}

func main() {
	me := env{make(map[string]value), &env{make(map[string]value), nil}}
	repl(&me)
	my_tree := tree{value_symbol_init(make([]rune, 0)), false, nil, nil}
	program := "(len (list (list 1 4) 2 3 4 5 6 7))"
	fmt.Println(program)
	parse([]rune(program), 0, &my_tree, make([]rune, 0))
	//print_tree(&my_tree)
	fmt.Println("\nEval: ")
	r, err := prognfunc(&my_tree, &env{make(map[string]value), nil})
	if err == nil {
		print_value(r)
	} else {
		fmt.Println(err)
	}
}
