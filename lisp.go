package main

import "fmt"

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

func main() {
	my_tree := tree { value { make([]rune, 0), nil },
		false,
		nil, nil}
	parse([]rune("((lambda (x) (+ x 2)) 5) (greek 'anna) 1 5"), 0, &my_tree)
	print_tree(&my_tree)
}
