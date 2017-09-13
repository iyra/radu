# radu

## What?
An implementation of language which shares many features similar to Common Lisp/Scheme, though it does not try to emulate either.

## What works (and a guide for those new to the Lisp language family)

* A value is anything in the language, for example `5` is a value, so is `(1 2)` and so is `(lambda (x) (+ x 3))` etc.; evaluating a value always produces a value. Some values evaluate to themselves, for example a number always evaluates to the same number.
* Mentioning a non-number, non-string value that isn't bound (such as by `let`, `lambda`, `define`) will try to find the value in the environment, and if it can't, it will give you an error.
* Doing `(x y)` will attempt to run the function `x` on the argument `y`, similarly with multiple arguments. So giving radu `(5 22)` is nonsensical - why? Because 5 isn't a function. A function can also take no arguments, in which case it will look like `(x)`.
* `(lambda (var1 var2) value)` will define an anonymous function which takes one or more arguments (in this case,  two); for example, for a function called "adder" which just adds two numbers, one might have a lambda like: `(lambda adder (x y) (+ x y))`. A lambda will just produce a function, so it's not much use on its own. Because it produces a function, you can use it like: `((lambda adder (x y) (+ x y)) 3 2) where 3 and 2 are the arguments. This will produce 5 in this example.
* `(define identifier value)` will define a variable to be accessed within the current scope but (hopefully) not outside it. User-defined  functions are actually `lambda`s, so you can name your functions like this. There is no separate way to define functions. You can use `define` to re-define things you've already defined.
* `(list a b c)` will create a list, in this case with three values but you can have more or less or even zero (`(list)`); each of the items is evaluated before the list is given to you. A list looks like `(a b c)` but do not mistake this for the function `a` calling the arguments `b` and `c`. It will only do that if you *evaluate* `(a b c)`. So `(eval (list my-function arg1 arg2))` will run `(my-function arg1 arg2)` as mentioned in the third bullet point.
* `(car my-list)` will get the first item of the list `my-list`. It only works on lists. If `my-list` were `(list a b c)` then `car` would return `a`
* `(cdr my-list)` will get the rest of a list; to use the list defined above again, it would produce `(b c)`
* `(succ number)` will return number+1; it's only valid for numbers, though.
* `(dofor my-function my-list)` is similar to foreach in other languages, but it applies a function to each element of `my-list` and returns the new list. For example, `(dofor succ (list 1 2 3))` will give you `(2 3 4)`.
* `(eq value1 value2)` will return `#t` (this means "True") if `value1` is equal to `value2`, and `#f` (meaning "False") otherwise. 
* `(+ number1 number2 ...)` will add numbers together and give their result. If all the numbers are integers it will produce an integer. If the numbers are a mix of integers and rationals (or they're just rationals) then it will produce a rational. If any of the numbers is a float, it will produce a float. There are three more arithmetic functions, `*`, `-` and `%` (mod) which do as you can guess. I haven't implemented `/` yet.
*     (if test-value
          thing-to-do-if-true
          thing-to-do-if-false)
  will evaluate `thing-to-do-if-true` if `test-value` returns the symbol `#t` (in this language it means "true") and it will do `thing-to-do-if-false` if it returns `#f` instead. For example, `(if (eq 3 (+ 4 1)) "yes" "no")` should give you the string "yes".
* `(progn value1 value2 ...)` will let you run one bit of code after the other. The values can be functions of course. The program you input on each line is automatically given to `progn` so if you give the input `(+ 4 2) (* 4 2)` then it will produce `8`, because you only see the result of the last thing you evaluate, but they really are all evaluated.
* `(let ((name 1 value1) (name2 value2) ...) my-function)` will bind values to names and then let you use those names in `my-function`. It is similar to `define`, but what it defines is local only. You can't access `name1` or `name2` outside it. For example, `(let ((x 3) (y 4)) (progn (+ x y) (* x y)))`
* `(nand bool1 bool2)` is the standard NAND operator; it will return `#t` if and only if both `bool1` and `bool2` are false. Using this you can make `not`, `and`, `or` etc. and combine these with `if` to get what's commonly found in other languages like `&&`, `|||` and more.
* `strcat`, `strindex`, `strlen` concatenate two or more string arguments, find the Nth character of a string and find the length of a string respectively. These should be Unicode-safe, so that the length of Ελλάδα for example should be 6, not the number of bytes in the string.
* `append` adds an item onto a list, for example `(append 6 (list 4 5))` will produce `(4 5 6)`. `prepend` does the same but adds to the front of the list instead.
* `(quote value)` will stop `value` from being evaluated.
* `(eval value)` will evaluate whatever it's given
* `(len list1)` will find the length of `list1`. For example, `(len (prepend 22.0 (list 1 4 17)))` will give you 4.
* `(quit)` or `(exit)` to leave radu.

## What doesn't work (but may in future)
* Quasiquoting
* Macros
* `cons` seems to be a little broken
* Functions only check that they have enough args, not if they have too many (except `lambda`'s argument list)
* Rationals calculations just haven't been implemented yet
* Newlines in input
* Quoting a tree currently does nothing but I hope to make it behave like `(list 'arg1 'arg2 ...)` if possible.

## Dependencies

Literally none, except for `go`; you can compile by doing `go build lisp.go`. The resulting binary is compatible with `gdb` if you need to do any debugging.

## Credits

Thanks to Ioannis Panagiotis Koutsidis for helping me sort out the behaviour of quote, the idea behind the parser, the environment structure, how to make let and lambda bindings non-persistent and how to make define bindings persistent only to their scope, and general suggestions and debugging advice.

## License

Copyright (C) Iyra Gaura 2017

Licensed under the CC0 Public Domain Dedication: https://creativecommons.org/publicdomain/zero/1.0/

You are free to use it as you like therefore, but I would appreciate an e-mail or mention if you use it for something cool.