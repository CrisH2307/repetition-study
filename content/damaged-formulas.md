# Damaged formulas in content/source.md

pdftotext flattens every display formula. Fix these by hand in `content/source.md`, then run
`python3 scripts/parse_pdf.py --sync content/source.md content/unit.json` and `go test ./content/`.
Page = PDF page; `source.md` has a `<!-- page N -->` marker before each.

| Page | Problem | Damage |
| --- | --- | --- |
| 1 | Ex 3.11 hint | Transience sum split over 5 lines (sum sign, numerator, denominator, `< ∞`) |
| 2 | Ex 3.2(a) | Parentheses lost in `P(T^X < T^Y)`; fraction λ1/(λ1+λ2) split |
| 2 | Ex 3.2(b) | `e^{-(λ1+λ2)}` and `(λ1+λ2)^4 / 4!` on separate lines, exponent flattened |
| 2 | Ex 3.2(c) | Conditional probability chain and fractions interleaved, partly unreadable |
| 2 | Ex 3.2(d) | Integrals: limits, integrand and `dt` on different lines; closing fraction split |
| 3 | Ex 3.5 | Generator G and P(t) matrices lost brackets; entries like 4/5 + (1/5)e^{-5t} scrambled |
| 3 | Ex 3.8(a) | Matrix G and `π = (1/38)(3,7,9,19)` split |
| 3 | Ex 3.8(c) | `(-G_{-4})^{-1} 1` subscript flattened; two matrices lost brackets |
| 3, 4 | Ex 3.11 | Sums with products `λ_{n-1}···λ_0 / β_n···β_1`; `λ_n = 1 + 1/(n+1)` fractions stacked |
| 4 | Ex 3.14 | Sums, `π0 = 1/e`, `π_n = 1/(n! e)` split across lines |
| 5 | Q1(b), (d) | Sums `Σ e^{-6}6^n/n!` with limits on other lines |
| 6 | Q2 | Compound sum, `E[Y]`, `√4680`, percentile fraction |
| 7 | Q3 | 3x3 generator; forward-equation fractions; `0.994^2` printed as `0.9942` |
| 8 | Q4 | 4x4 generator, `π = (1/8)(1,3,2,2)`, inverse matrices |
| 9 | Q5 | 4x4 generator with 1/11 entries; `π0 = 1/6.2`; `lim p_{i0}(t)` subscript |
| 10 | Q6 | Integrals, `P_B` algebra, `E[T]` fractions, line `λ²_2` floating above |
| 11 | Q7 | Piecewise `λ_n` (brace lost), tridiagonal generator with dots, recursion fractions, powers |
| 12 | Q7 | Series `Σ (1/2)^n / n`, `ln` terms, `π0` fraction |
| 13 | Q8 | 3x3 generators, `π = (1/37)(16,12,9)`, fractions 28/37, 88/97 |

## Typos in the PDF itself (not parser damage)
- p.6 Q2: `E[Y] = E[N(13)]E[X] = 13·50·7.2 = 1625` should use E[X] = 2.5 (13·50·2.5 = 1625). Also `P(X ≤ Q90)` should be Y. Q2 is not in the claim list for this reason.
- p.3 Ex 3.8(a), p.8 Q4(a): "with π1 = 1" fixes the scale while solving. The stated π is then the normalised vector. Claims state only the normalised vector.
- p.11 Q7: "with π1 = 1" but the solution then writes everything in terms of π0.
