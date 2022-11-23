# Lark

**Lark** is a Lisp interpreter aiming to implement Scheme.

## Basics

``` scheme
(define pi 3.14)
(define r 10)
(define square (lambda (x) (* x x)))
(define area (lambda (r) (* pi (square r))))
(area r)
;; 314
```

