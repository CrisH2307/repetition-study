# P0-EVALDATA step 1: 4 concepts, 16 required claims (approved by Cris)

Claims are in our own words, not copied from the PDF. Each is a definition (D),
statement (S), condition (C) or method (M), never a worked numeric answer.
Locators use the existing `p.N, Ex 3.x` / `p.N, Qn` form checked by
`content/unit_test.go`. Claim ids are reused, so `content/seed.sql` still works.
The other 4 concepts in `content/unit.json` stay as they are.

## 101 Exponential races
1. (S) If T1 ~ Exp(λ1) and T2 ~ Exp(λ2) are independent, then P(T1 < T2) = λ1/(λ1+λ2). `p.2, Ex 3.2`
2. (S) The minimum min(T1, T2) of two independent exponential times is Exp(λ1+λ2). `p.3, Ex 3.3 (cont.)`
3. (M) P(T1 < T2) is derived by conditioning on T1 = t, using P(T2 > t) = e^{-λ2 t}, and integrating against the density λ1 e^{-λ1 t}. `p.10, Q6`
4. (D) An Exp(λ) time has mean 1/λ, so a stated mean m converts to the rate 1/m. `p.5, Q1`

## 102 Superposition of Poisson processes
1. (S+C) If X(t) and Y(t) are independent Poisson processes with rates λ1 and λ2, then X(t)+Y(t) is a Poisson process with rate λ1+λ2. `p.2, Ex 3.2`
2. (D) In a Poisson process of rate λ, the count in time t is Poisson(λt): P(N(t)=k) = e^{-λt}(λt)^k/k!. `p.2, Ex 3.2`
3. (M) To get P(X(1)=k | X(1)+Y(1)=k), rewrite the event as {X(1)=k, Y(1)=0}, split it by independence, and divide by the Poisson(λ1+λ2) probability of the total. `p.2, Ex 3.2`
4. (M) For an independent random time T, P(X(T)=k) is found by integrating P(X(t)=k) against the density of T. `p.2, Ex 3.2`

## 105 Stationary distribution from balance equations
1. (D) The stationary distribution π of a finite CTMC with generator G is a row vector solving πG = 0. `p.3, Ex 3.8`
2. (C) πG = 0 only fixes π up to scale; it must be normalised so its entries sum to 1. `p.8, Q4`
3. (S) The long-run proportion of time the chain spends in state i is π_i. `p.11, Q7`
4. (S) The sojourn time in state i is exponential with rate -g_ii (minus the diagonal entry), so its mean is 1/(-g_ii). `p.3, Ex 3.8`

## 107 Birth-death recurrence criteria
1. (C) A birth-death process is positive recurrent, equivalently has a stationary distribution, if and only if the sum over n >= 1 of (λ_{n-1}···λ_0)/(β_n···β_1) is finite. `p.3, Ex 3.11`
2. (S) When that sum S is finite, π0 = 1/(1+S) and π_n = π0 (λ_{n-1}···λ_0)/(β_n···β_1). `p.4, Ex 3.14`
3. (C) If that sum and the sum over n >= 1 of (β_n···β_1)/(λ_n···λ_1) both diverge, the process is null recurrent. `p.4, Ex 3.11 (cont.)`
4. (M) To classify a given process, simplify the products (they often telescope) and test the resulting series for convergence. `p.4, Ex 3.11 (cont.)`

Unverified: the transient criterion (second sum finite) is not stated in the PDF, so it is left out.
Eval plan after approval: 5 explanations per concept x 4 claims = 80 claim labels.
