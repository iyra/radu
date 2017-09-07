package main

import "fmt"
import "strconv"
import "errors"
import "unicode"

const (
	error_noconv_int = iota
	error_least2_args = iota
	num_float = iota
	num_int = iota
	num_undef = iota
	symbol_type = "sym"
	int_type = "int"
	float_type = "float"
	number_type = "number"
)

type number_value struct {
	number_type int
	floatval float64
	intval int64
}

type value struct {
	symbol []rune
	ast *tree
}
type tree struct {
	val value
	done_val bool
	next *tree
	parent *tree
}

type convError struct {
	from string
	to string
}

func (e *convError) Error() string {
    return fmt.Sprintf("convError: Can't convert %s to %s", e.from, e.to)
}

func show_value(valtype string, val string) string {
	if(val != "") {
		return fmt.Sprintf("[%s %s]", valtype, val)
	} else {
		return fmt.Sprintf("[%s]", valtype)
	}
}

func all_digits_p(symbol rune) bool {
	for i,e := range symbol {
		if !unicode.IsDigit(e) {
			return false
		}
	}
	return true
}

func get_number(symbol []rune) (number_value, error) {
	// integer
	if all_digits_p(symbol) {
		if r, err := strconv.ParseInt(string(symbol), 64); err == nil {
			return number_value { num_int, 0, r }, nil
		} else {
			return number_value { num_undef, 0, 0 }, &convError{show_value(symbol_type, string(symbol)), show_value(int_type, "")}
		}
	}

	// float
	if strings.Contains(string(symbol), ".")
	&& strings.Count(string(symbol), ".") == 1 {
		// contains a single .
		if r, err := strconv.ParseFloat(string(symbol), 64); err == nil {
			return number_value { num_float, r, 0 }, nil
		} else {
			return number_value { num_undef, 0, 0 }, &convError{show_value(symbol_type, string(symbol)), show_value(float_type, "")}
		}
	}

	return number_value { num_undef, 0, 0 }, &convError{show_value(symbol_type, string(symbol)), show_value(number_type, "")}
}

func parse(input []rune, n int, ast *tree) int{
	if n == len(input) {
		fmt.Printf("done")
	} else {
		switch c := string(input[n]); c {
		case "(":
			if ast.done_val {
				// value has finished collecting
				// now collect arguments
				//tree { value { make([]rune, 0), nil }, false, nil};
				// just move on because we do the tree allocation and nexting with ' '
				parse(input, n+1, ast)
			} else {
				if len(ast.val.symbol) == 0 {
					// case like ((... so parse
					ast.val.ast = &tree {value { make([]rune, 0), nil }, false, nil, ast};
					parse(input, n+1, ast.val.ast)
				} else {
					fmt.Printf("error: unexpected ( in tree value\n")
				}
			}
			break
		case ")":
			ast.done_val = true
			if(ast.parent != nil){
				return parse(input, n+1, ast.parent)
			} 
			break
		case " ":
			if !ast.done_val {
				ast.done_val = true
			}

			ast.next = &tree {value { make([]rune, 0), nil } ,false, nil, ast.parent}
			
			if input[n+1] != ' ' {
				// get next argument
				parse(input, n+1, ast.next)
			} else {
				for g := n; g < len(input); g++ {
					if input[g] != ' ' {
						parse(input, g, ast.next)
						break;
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

func print_tree(ast *tree){
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

func addfunc(ast *tree) (value, error) {
	if len(ast.value.symbol) == 0 || len(ast.next.value.symbol) == 0 {
		return error_least2_args
	}
	nlist := make([]inteface{}, 0)
	if v,err := get_number(ast.value.symbol); err == nil {
		nlist = append(nlist, v)
	} else {
		return value { make([]rune, 0), nil }, err
	}

	if v,err = get_number(ast.next.value.symbol); err == nil {
		nlist = append(nlist, v)
	} else {
		return value { make([]rune, 0), nil }, err
	}

	for i,e := range nlist {
		???
	}
}

func eval(ast *tree){
	if ast.val.ast == nil {
		switch string(ast.val.symbol) {
		case "+": return addfunc(ast.next)
			break;
		case "-": return subfunc(ast.next)
			break;
		default: fmt.Printf("unrecognised function: %s\n" % ast.val.symbol)
		}
	} else {
		eval(ast.val.ast)
	}
}

func main() {
	my_tree := tree { value { make([]rune, 0), nil },
		false,
		nil, nil}
	parse([]rune("((lambda (x) (+ x 2)) 5) (greek 'anna) 1 5"), 0, &my_tree)
	print_tree(&my_tree)
}
