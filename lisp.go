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
)

type number_value struct {
	floatval float64
	intval   int64
}

type value struct {
	valtype int
	symbol  []rune
	ast     *tree
	number  number_value
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

// func get_number(symbol []rune) (number_value, error) {
// 	// integer
// 	if all_digits_p(symbol) {
// 		if r, err := strconv.ParseInt(string(symbol), 64); err == nil {
// 			return number_value { num_int, 0, r }, nil
// 		} else {
// 			return number_value { num_undef, 0, 0 }, &convError{show_value(symbol_type, string(symbol)), show_value(int_type, "")}
// 		}
// 	}

// 	// float
// 	if strings.Contains(string(symbol), ".")
// 	&& strings.Count(string(symbol), ".") == 1 {
// 		// contains a single .
// 		if r, err := strconv.ParseFloat(string(symbol), 64); err == nil {
// 			return number_value { num_float, r, 0 }, nil
// 		} else {
// 			return number_value { num_undef, 0, 0 }, &convError{show_value(symbol_type, string(symbol)), show_value(float_type, "")}
// 		}
// 	}

// 	return number_value { num_undef, 0, 0 }, &convError{show_value(symbol_type, string(symbol)), show_value(number_type, "")}
// }

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
					ast.val.ast = &tree{value{t_head_symbol, make([]rune, 0), nil, number_value{0, 0}}, false, nil, ast}
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

			ast.next = &tree{value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}, false, nil, ast.parent}
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
		fmt.Printf(string(ast.val.symbol))
		//fmt.Printf("[%p]", ast.parent)
		fmt.Printf("[%d]", ast.val.valtype)
	} else {
		fmt.Printf("(")
		print_tree(ast.val.ast)
		fmt.Printf(")")
	}
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
	return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}
}

func quotefunc(ast *tree, bindings *env) (value, error) {
	if ast.next != nil && argcount(ast, 0) == 1 {
		switch ast.next.val.valtype {
		case t_symbol:
			return ast.next.val, nil
			break
		case t_tree:
			return ast.next.val, nil
			break
		case t_number_float, t_number_int, t_number_rational:
			return ast.next.val, nil
		}
	}
	return blank_value(), errors.New("usage: (quote <value>)")
}

//func listeval(ast *tree, bindings *env) (value, error) {

//}

func listfunc(ast *tree, bindings *env, orig *tree) (value, error) {
	/* (list x y z) => (eval(x) -> eval(y) -> eval(z)) */
	if ast.next == nil {
		return orig.val, nil
	}
	var err error
	ast.next.val, err = eval(ast.next, bindings)
	if err != nil {
		return blank_value(), err
	}
	return listfunc(ast.next, bindings, orig)
}

func consfunc(ast *tree, bindings *env, orig *tree) (value, error) {
	return blank_value(), nil
}

func lambdafunc(ast *tree, bindings *env) (value, error) {
	/* ((lambda (x) (fn x)) y)
	((lambda -> (x) -> (fn -> x)) -> y)
	*/
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
	return strconv.ParseInt(string(symbol), 10, 64)
}

func is_float(symbol []rune) bool {
	if strings.Count(string(symbol), ".") == 1 {
		for _, e := range symbol {
			if !unicode.IsDigit(e) && e != '.' {
				return false
			}
		}
	}
	return false
}

func conv_float(symbol []rune) (float64, error) {
	return strconv.ParseFloat(string(symbol), 64)
}

func bound(symbol []rune, bindings *env) (value, error) {
	if val, ok := bindings.values[string(symbol)]; ok {
		return val, nil
	}
	if bindings.prev != nil {
		return bound(symbol, bindings.prev)
	} else {
		return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}},
			errors.New(fmt.Sprintf("error: symbol %s not found in environment.", string(symbol)))
	}
}

/*func eval(ast *tree, bindings *env) (value, error) {
	return blank_value(), nil
}*/

func eval(ast *tree, bindings *env) (value, error) {
	switch ast.val.valtype {
	case t_head_symbol:
		// like (x y z), we are talking about x
		// apply to args
		switch sym := string(ast.val.symbol); sym {
		case "quote":
			return quotefunc(ast, bindings)
		case "cons":
			return consfunc(ast, bindings, ast)
		case "list":
			return listfunc(ast, bindings, ast)
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
		if is_integer(rsym) {
			if v, err := conv_integer(rsym); err == nil {
				return value{t_number_int, make([]rune, 0), nil, number_value{0, v}}, nil
			} else {
				return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}, err
			}
		}
		if is_float(rsym) {
			if v, err := conv_float(rsym); err == nil {
				return value{t_number_float, make([]rune, 0), nil, number_value{v, 0}}, nil
			} else {
				return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}, err
			}
		}
	case t_number_float, t_number_int, t_number_rational:
		return ast.val, nil
	}
	return eval(ast.val.ast.next, bindings)
}

// func eval(ast *tree, bindings *env) (value, error) {
// 	if ast.val.ast == nil {
// 		if res, finderr := bound(ast.val.symbol, bindings); finderr == nil {
// 			// symbol found
// 		} else {
// 			rsym := ast.val.symbol
// 			switch sym := string(rsym); sym {
// 			case "quote":
// 				return quotefunc(ast, bindings)
// 				break
// 			case "cons":
// 				return consfunc(ast, bindings)
// 				break
//             case "list":
//                 return listfunc(ast, bindings, ast)
//                 break
// 			default:
// 				if is_integer(rsym) {
// 					if v, err := conv_integer(rsym); err == nil {
// 						return value{t_number_int, make([]rune, 0), nil, number_value{0, v}}, nil
// 					} else {
// 						return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}, err
// 					}
// 				}
// 				if is_float(rsym) {
// 					if v, err := conv_float(rsym); err == nil {
// 						return value{t_number_float, make([]rune, 0), nil, number_value{v, 0}}, nil
// 					} else {
// 						return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}, err
// 					}
// 				}
// 				// ...
// 				return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}, finderr
// 			}
// 		}
// 	}
// 	return eval(ast.val.ast, bindings)
// }

func main() {
	my_tree := tree{value{t_symbol, make([]rune, 0), nil, number_value{0, 0}},
		false,
		nil, nil}
	program := "((lambda (x) (+ x 2)) 5) (greek 'anna) 1 5"
	fmt.Println(program)
	parse([]rune(program), 0, &my_tree)
	print_tree(&my_tree)
}
