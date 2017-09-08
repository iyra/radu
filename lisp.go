package main

import "fmt"
import "strconv"
import "errors"
import "unicode"

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
	values map[[]rune]value
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
					ast.val.ast = &tree{value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}, false, nil, ast, nil}
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

func quotefunc(ast *tree) value {
	if ast.val.ast == nil {
		// quoting a symbol like (quote x)
		return ast.symbol
	} else {
		// quoting an ast like (quote (+ 3 2))
		return ast.next
	}
	return value{t_symbol, make([]rune, 0), ast, number_value{0, 0}}
}

func nth_rune(str string, n int) (rune, err) {
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
		for i, e := range symbol {
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
	if val, ok := bindings.values[symbol]; ok {
		return val, nil
	}
	if env.prev != nil {
		return bound(symbol, env.prev), nil
	} else {
		return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}},
			errors.New(fmt.sprintf("error: symbol %s not found in environment." % string(symbol)))
	}
}

func eval(ast *tree, bindings *env) (value, err) {
	if ast.val.ast == nil {
		if res, finderr := bound(ast.val.symbol, bindings); finderr == nil {
			// symbol found
		} else {
			switch sym := ast.val.symbol; sym {
			case "quote":
				return quotefunc(ast.next)
				break
			default:
				if is_integer(sym) {
					if v, err := conv_integer(sym); err == nil {
						return value{t_number_int, make([]rune, 0), nil, number_value{}}, nil
					} else {
						return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}, err
					}
				}
				if is_float(sym) {

				}
				// ...
				return value{t_symbol, make([]rune, 0), nil, number_value{0, 0}}, finderr
			}
		}
	} else {
		eval(ast.val.ast)
	}
}

func main() {
	my_tree := tree{t_symbol, value{make([]rune, 0), nil, number_value{0, 0}},
		false,
		nil, nil}
	parse([]rune("((lambda (x) (+ x 2)) 5) (greek 'anna) 1 5"), 0, &my_tree)
	print_tree(&my_tree)
}
