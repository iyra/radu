package main

import "fmt"
import "strconv"
import "errors"
import "unicode"
import "strings"

const (
	error_noconv_int  = iota
	error_least2_args = iota
	num_float         = iota
	num_int           = iota
	num_undef         = iota
	t_symbol          = iota
	t_tree            = iota
	t_number_float    = iota
	t_number_int      = iota
	t_number_rational = iota
	t_head_symbol     = iota
	t_function        = iota
)

var typenames = map[int]string{
	t_symbol:          "symbol",
	t_tree:            "tree",
	t_number_float:    "float",
	t_number_int:      "int",
	t_number_rational: "rational",
	t_head_symbol:     "head-symbol",
	t_function:        "function",
}

type number_value struct {
	floatval float64
	intval   int64
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
	valtype  int
	symbol   []rune
	ast      *tree
	number   number_value
	function function_value
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
	return value{t_symbol, name, nil, number_value{0, 0}, function_value{make([][]rune, 0), nil}}
}

func value_head_symbol_init(name []rune) value {
	return value{t_head_symbol, name, nil, number_value{0, 0}, function_value{make([][]rune, 0), nil}}
}

func value_ast_init(ast *tree) value {
	return value{t_tree, make([]rune, 0), ast, number_value{0, 0}, function_value{make([][]rune, 0), nil}}
}

func value_number_int_init(n int64) value {
	return value{t_number_int, make([]rune, 0), nil, number_value{0, n}, function_value{make([][]rune, 0), nil}}
}

func value_number_float_init(n float64) value {
	return value{t_number_float, make([]rune, 0), nil, number_value{n, 0}, function_value{make([][]rune, 0), nil}}
}

func value_function_init(args [][]rune, action *tree) value {
	return value{t_function, make([]rune, 0), nil, number_value{0, 0}, function_value{args, action}}
}

func parse(input []rune, n int, ast *tree) int {
	if n == len(input) {
		fmt.Printf("done")
	} else {
		switch c := input[n]; c {
		case '(':
			if ast.done_val {
				// value has finished collecting
				// now collect arguments
				//tree { value { make([]rune, 0), nil }, false, nil};
				// just move on because we do the tree allocation and nexting with ' '
				parse(input, n+1, ast)
			} else {
				if len(ast.val.symbol) == 0 {
					// case like ((... so parse
					ast.val.valtype = t_tree
					ast.val.ast = &tree{value_head_symbol_init(make([]rune, 0)), false, nil, ast}
					parse(input, n+1, ast.val.ast)
				} else {
					fmt.Printf("error: unexpected ( in tree value\n")
				}
			}
			break
		case ')':
			ast.done_val = true
			if ast.parent != nil {
				return parse(input, n+1, ast.parent)
			}
			break
		case ' ':
			if !ast.done_val {
				ast.done_val = true
			}

			ast.next = &tree{value_symbol_init(make([]rune, 0)), false, nil, ast.parent}
			if input[n+1] != ' ' {
				// get next argument
				parse(input, n+1, ast.next)
			} else {
				for g := n; g < len(input); g++ {
					if input[g] != ' ' {
						parse(input, g, ast.next)
						break
					}
				}
			}
			break
		default:
			ast.val.symbol = append(ast.val.symbol, input[n])
			parse(input, n+1, ast)
		}
	}
	return 0
}

func print_tree(ast *tree) {
	if ast.val.ast == nil {
		print_value(ast.val)
	} else {
		fmt.Printf("(")
		print_tree(ast.val.ast)
		fmt.Printf(")")
	}
	fmt.Printf("[%s]", typenames[ast.val.valtype])
	if ast.next != nil {
		fmt.Printf(" -> ")
		print_tree(ast.next)
	}

}

/*
(+ 3 2)
((x) y)
(p z)*/

func blank_value() value {
	return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}, function_value{make([][]rune, 0), nil}}
}

func quotefunc(ast *tree, bindings *env) (value, error) {
	if ast.next != nil {
		return value_ast_init(&tree{ast.next.val, false, nil, nil}), nil
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
		return nil, errors.New("usage: (list x[ y z]); members will be evaluated.")
	}
	if r, err := listeval(ast.next, bindings, ast.next); err == nil {
		print_value(value{t_tree, make([]rune, 0), r, number_value{0, 0}, function_value{make([][]rune, 0), nil}})

		return &tree{value_ast_init(r), true, nil, nil}, nil
	} else {
		return nil, err
	}
}

func consfunc(ast *tree, bindings *env, orig *tree) (value, error) {
	return blank_value(), nil
}

func is_symbol(v value) bool {
	return v.valtype == t_head_symbol || v.valtype == t_symbol
}

func lambda_arglist(ast *tree, agg [][]rune, bindings *env) ([][]rune, error) {
	print_tree(ast)
	p, err := eval2(ast, bindings)
	if err != nil {
		return make([][]rune, 0), err
	}
	print_value(p)
	if !is_symbol(p) {
		fmt.Println("prob")
		return make([][]rune, 0), errors.New(fmt.Sprintf("error: lambda arglist must contain symbols only, given %s", typenames[ast.val.valtype]))
	}
	fmt.Println("PUSHING ", string(p.symbol))
	if ast.next == nil {
		return append(agg, p.symbol), nil
	} else {
		return lambda_arglist(ast.next, append(agg, p.symbol), bindings)
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
	fmt.Println("converting ", string(symbol))
	return strconv.ParseInt(string(symbol), 10, 64)
}

func is_float(symbol []rune) bool {
	if strings.Count(string(symbol), ".") == 1 {
		fmt.Printf("there is only one . in %s", string(symbol))
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
	}*/
	if val, ok := bindings.values[string(symbol)]; ok {
		return val, nil
	}
	if bindings.prev != nil {
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
		if g.valtype == t_number_int || g.valtype == t_number_float {
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
		if e.valtype == t_number_rational {
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
	return t_number_rational
}

func addfunc(ast *tree, bindings *env) (value, error) {
	vlist, err := collect_number_values(ast.next, bindings, make([]value, 0))
	if err == nil {
		var total float64 = 0
		for _, e := range vlist {
			if e.valtype == t_number_float {
				total += e.number.floatval
			}
			if e.valtype == t_number_int {
				total += float64(e.number.intval)
			}
		}
		return value_number_float_init(total), nil
	} else {
		return blank_value(), err
	}
}

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
	print_value(v)

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
		fmt.Printf("len get_subjects = %d, len func.args = %d\n", len(g), len(v.function.args))
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
		/*fmt.Printf("GGG ")
		print_value(v.ast.val.ast.val)*/
		return v.ast.val.ast.val, nil
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
		if v.ast.val.ast.next == nil {
			return blank_value(), errors.New("error: can only cdr a list with more than one value")
		}
		fmt.Printf("GGV ")
		print_value(value_ast_init(&tree{value_ast_init(v.ast.val.ast.next), true, nil, nil}))
		return value_ast_init(&tree{value_ast_init(v.ast.val.ast.next), true, nil, nil}), nil
	} else {
		return blank_value(), e
	}
}

func let_binds(b *tree, names [][]rune, values []value, bindings *env) ([][]rune, []value, error) {
	fmt.Println("from let_binds: ")
	if b == nil {
		for _, n := range names {
			fmt.Println(string(n))
		}
		for _, v := range values {
			print_value(v)
			fmt.Println()
		}
		return names, values, nil
	}
	if b.val.valtype == t_tree {
		print_tree(b.val.ast)
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
	fmt.Println("binds:")
	print_tree(ast.next)
	fmt.Println("function:")
	print_tree(ast.next.next)
	fmt.Println("bind_let:")
	if v, e := bind_let(ast.next, bindings); e == nil {
		fmt.Println("letfunc evaling")
		fmt.Println("ast.next.next is ")
		print_tree(ast.next.next)
		return eval2(ast.next.next, v)
	} else {
		return blank_value(), e
	}

}

func eval2(ast *tree, bindings *env) (value, error) {
	fmt.Println("CALL")
	/* (x[*] y[*] z[*] ...)[tree]
		x must either be (1) builtin [val symbol] or (2) eval to a [val function_value]

	s
	s must either be (1) arbitrary [val symbol] (2) eval to a number (3) be a number
	*/
	if ast.val.valtype == t_tree {
		print_tree(ast)
		fmt.Printf("valtype of")
		print_value(ast.val)
		fmt.Printf(" is %s\n", typenames[ast.val.valtype])
		//print_tree(ast.val.ast)
		if ast.val.ast.val.valtype == t_symbol || ast.val.ast.val.valtype == t_head_symbol {
			fmt.Printf("ast.val.ast.val.valtype of %s is symbol/head symbol\n", string(ast.val.ast.val.symbol))
			switch sym := string(ast.val.ast.val.symbol); sym {
			case "quote":
				fmt.Println("doing quote")
				return quotefunc(ast.val.ast, bindings)
			case "cons":
				return consfunc(ast.val.ast, bindings, ast)
			case "list":
				if p, e := listfunc(ast.val.ast, bindings, ast); e == nil {
					return value_ast_init(p), nil
				} else {
					return blank_value(), e
				}
			case "succ":
				return succfunc(ast.val.ast, bindings)
			case "+":
				return addfunc(ast.val.ast, bindings)
			case "lambda":
				return lambdafunc(ast.val.ast, bindings)
			case "eval":
				return evalfunc(ast.val.ast, bindings)
			case "car":
				return carfunc(ast.val.ast, bindings)
			case "cdr":
				return cdrfunc(ast.val.ast, bindings)
			case "let":
				return letfunc(ast.val.ast, bindings)
			default:
				fmt.Println("looking for ", string(ast.val.ast.val.symbol))
				if res, finderr := bound(ast.val.ast.val.symbol, bindings); finderr == nil {
					return eval2(&tree{value_ast_init(&tree{res, true, ast.val.ast.next, nil}), true, nil, nil}, bindings)
				} else {
					return blank_value(), finderr // couldn't find the x in (x y)
				}
			}
		}
		if g, err := eval2(ast.val.ast, bindings); err == nil && g.valtype == t_function {
			fmt.Println("performing -- ")
			print_tree(ast.val.ast.next)
			return performfunc(g, bindings, ast.val.ast.next)
		}
		return blank_value(), errors.New("usage: (x [a b c ...]) where x is a builtin or lambda")
	}

	if ast.val.valtype == t_symbol || ast.val.valtype == t_head_symbol {
		rsym := ast.val.symbol
		// like the y or z in (x y z)
		// or any other symbol sent to evaluate
		//symbols evaluate to numbers if they are numbers
		fmt.Println("rsym is ", string(ast.val.symbol))
		fmt.Println(is_integer(rsym), is_float(rsym))
		if is_integer(rsym) {
			if v, err := conv_integer(rsym); err == nil {
				fmt.Println("returning int")
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

		if res, finderr := bound(ast.val.symbol, bindings); finderr == nil {
			return res, nil
		}

		// it's not a number so just return it
		return ast.val, nil
	}

	if ast.val.valtype == t_number_int || ast.val.valtype == t_number_float || ast.val.valtype == t_function {
		return ast.val, nil
	}
	if ast.next != nil {
		return eval2(ast.next, bindings)
	}
	return blank_value(), errors.New("nothing to do")
}

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

func main() {
	my_tree := tree{value_symbol_init(make([]rune, 0)), false, nil, nil}
	program := "(let ((x (lambda (p z) (+ p 2 z))) (h 19)) (quote (+ (x h 4) h)))"
	fmt.Println(program)
	parse([]rune(program), 0, &my_tree)
	//print_tree(&my_tree)
	fmt.Println("\nEval: ")
	r, err := eval2(&my_tree, &env{make(map[string]value), nil})
	if err == nil {
		print_value(r)
	} else {
		fmt.Println(err)
	}
}
