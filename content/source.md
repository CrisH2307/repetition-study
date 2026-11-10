<!-- page 1 -->
                          MATH 4430/6602 STOCHASTIC PROCESSES
                                   PRACTICE PROBLEM SET 3


Practice Problems
The following practice problems are from the book “Lawler, Gregory F. Introduction to stochastic
processes. Second edition. Chapman & Hall/CRC, Boca Raton, FL, 2006.”
   Chapter 3: 3.2, 3.3, 3.5, 3.8, 3.11, 3.14
   For Exercise 3.11, you can use the fact that a birth-death process with birth rates {λn }n=0,1,2
and death rates {βn }n=1,2,3 is transient if and only if
                                      ∞
                                      X βn · · · β2 β1
                                                             < ∞.
                                      n=1
                                            λn · · · λ2 λ1

In addition to the exercises in the book, some additional problems are given in this problem set.
Please submit:

    • Additional practice problems: Q1, Q3, Q4, Q7, Q8.




                                                   1

<!-- page 2 -->
1. (Solution to Ex 3.2) (a) Let T X and T Y be the times when the first customer arrives in
   store 1 and 2, respectively. We know that T X ∼ Exp(λ1 ), T Y ∼ Exp(λ2 ), and they are
   independent. Then the probability that the first customer arrives in store 1 equals
                                                            λ1
                                     P TX < TY =
                                              
                                                                 .
                                                         λ1 + λ2

   (b) By Proposition 1.1 in lecture note 3, X(t) + Y (t) is a Poisson process with rate λ1 + λ2 .
   Hence, the probability that in the first hour, a total of exactly four customers have arrived at
   the two stores equals

                                                                    (λ1 + λ2 )4
                          P (X(1) + Y (1) = 4) = e−(λ1 +λ2 )                    .
                                                                        4!

   (c) We have

                                             P (X(1) = 4, X(1) + Y (1) = 4)
      P (X(1) = 4| X(1) + Y (1) = 4) =
                                                   P (X(1) + Y (1) = 4)
                                             P (X(1) = 4, Y (1) = 0)
                                           =
                                              P (X(1) + Y (1) = 4)
                                             P (X(1) = 4) P (Y (1) = 0)
                                           =                            , by independence
                                                P (X(1) + Y (1) = 4)
                                                         λ4
                                                    e−λ1 4!1 e−λ2
                                           =                         4   , by part (b)
                                             e−(λ1 +λ2 ) (λ1 +λ
                                                             4!
                                                                2)

                                                         4
                                                  λ1
                                           =                  .
                                               λ1 + λ2

   (d) Since T ∼ Exp(λ2 ), we have
                                               Z ∞
                        P (X(T ) = k) =              P (X(t) = k) fT (t)dt
                                                0
                                               Z ∞
                                                             (λ1 t)k
                                          =          e−λ1 t          λ2 e−λ2 t dt
                                                0              k!
                                               λk1 λ2 ∞ −(λ1 +λ2 )t k
                                                      Z
                                          =                  e           t dt
                                                 k! 0
                                                                    Z ∞
                                                      λk1 λ2
                                          =                    k+1
                                                                         e−x xk dx
                                               k!(λ1 + λ2 )          0
                                                        k
                                                      λ1 λ2
                                          =                        k!
                                               k!(λ1 + λ2 )k+1
                                                                      k
                                                   λ2            λ1
                                          =                                .
                                               λ1 + λ2 λ1 + λ2

2. (Solution to Ex 3.3) (a) By Proposition 1.1 in lecture note 3, Zt is a Poisson process with rate
   λ1 + λ2 .

                                               2

<!-- page 3 -->
   (b) Let T X and T Y be the times when the first call arrives in process X and Y , respectively. We
   know that T X ∼ Exp(λ1 ), T Y ∼ Exp(λ2 ), and they are independent. Then the probability
   that the first call comes on the first phone equals
                                                                 λ1
                                     P TX < TY =
                                              
                                                                      .
                                                              λ1 + λ2

   (c) We denote by T := min(T X , T Y ) the first time that at least one call has come from one
   of the two phones. It is easy to see that T follows Exp(λ1 + λ2 ).

3. (Solution to Ex 3.5) Note α(i, j) is qij in the lecture note. Since the generator
                                                           
                                                  −1 1
                                        G=                    ,
                                                   4 −4

   by Example 2.1 in the lecture note, the transition matrix is given by
                                           4 1 −5t 1 1 −5t 
                                    tG        5
                                                + 5e      5
                                                            − 5e
                           P (t) = e =             4 −5t 1              .
                                              4
                                              5
                                                − 5e      5
                                                            + 45 e−5t

4. (Solution to Ex 3.8) (a) Solve the balance equation
                                                      
                                    −3 1       1    1
                                 0 −3 2            1 
                       πG = π                          = 0, with π1 = 1
                                 1       2 −4 1 
                                     0    0    1 −1

   we obtain the stationary distribution π = 38
                                             1
                                                (3, 7, 9, 19).
   (b) The sojourn time of state 1 U1 ∼ Exp(3). Hence, given X(0) = 1, the expected amount
   of time until it changes state for the first time is given by
                                                       1
                                              E [U1 ] = .
                                                       3

   (c) Since                                  −1     
                                       3 −1 −1          1
                               −1
                        (−G−4 ) 1 =   0 3 −2     1=  1 ,
                                      −1 −2 4           1
   given the chain starts in state 1, the expected amount of time until the chain is in state 4 is
   equal to 1.

5. (Solution to Ex. 3.11) Recall Example 2.4 in lecture note 3. The necessary and sufficient
   condition for the existence of the stationary distribution, or equivalently the birth-death
   process is positive recurrent, is that
                                      ∞
                                      X λn−1 · · · λ1 λ0
                                                               < ∞.
                                       n=1
                                             βn · · · β2 β1


                                                3

<!-- page 4 -->
   For the birth-death process with birth rate λn = 1 + n+1
                                                          1
                                                            = n+2
                                                              n+1
                                                                  and death rate βn = 1, we
   obtain         ∞                     ∞                     ∞
                 X   λn−1 · · · λ1 λ0 X n + 1 n            2 X
                                     =                 ··· =     (n + 1) = ∞,
                 n=1
                      βn · · · β2 β1   n=1
                                             n n−1         1 n=1
   and                ∞                            ∞                           ∞
                      X βn · · · β2 β1             X           1               X    2
                                               =       n+2 n+1             =           = ∞.
                      n=1
                              λn · · · λ2 λ1       n=1 n+1 n
                                                               · · · 23        n=1
                                                                                   n+2
   So it is null-recurrent.
   For the birth-death process with birth rate λn = 1 − n+2
                                                         1
                                                            = n+1
                                                              n+2
                                                                  and death rate βn = 1, we
   obtain          ∞                     ∞                     ∞
                  X   λn−1 · · · λ1 λ0 X n n − 1           1 X 1
                                      =                ··· =            = ∞,
                  n=1
                       βn · · · β2 β1   n=1
                                            n+1 n          2 n=1 n + 1
   and                ∞                            ∞                           ∞
                      X βn · · · β2 β1             X           1               X n+2
                                               =       n+1 n               =             = ∞.
                      n=1
                              λn · · · λ2 λ1       n=1 n+2 n+1
                                                               · · · 23        n=1
                                                                                     2
   So it is null-recurrent.

6. (Solution to Ex 3.14) Since
                                ∞                             ∞
                                X λn−1 · · · λ1 λ0            X 1
                                                          =              = e − 1 < ∞,
                                 n=1
                                       βn · · · β2 β1         n=1
                                                                  n!

   this birth-death process is positive recurrent. By Example 2.4 in lecture note 3, the stationary
   distribution is given by
                                                    1            1
                                   π0 =       P∞ λn−1 ···λ1 λ0 =
                                          1 + n=1 β ···β β       e
                                                                   n     2 1

   and
                                                                1
                        λn−1 · · · λ1 λ0      1
                 πn =                                         = n!
                                                                   , for n = 1, 2, 3, . . .
                         βn · · · β2 β1 1 + ∞   λn−1 ···λ1 λ0   e
                                           P
                                            n=1 β ···β β      n    2 1




                                                         4

<!-- page 5 -->
                 Additional Practice Problems for CTMC
1. Arrivals of customers into a local board game store follow a Poisson process with rate λ = 20
   arrivals per hour. Suppose that the probability that a customer buys some board games (one
   sale) is p = 0.30.
   (a) Find the expected number of sales made during an eight-hour business day.
   (b) Find the probability that 10 or more sales are made in one hour.
   (c) Find the expected time of the first sale of a day. If the store opens at 10 a.m.
   (d) Given 6 sales are made in one hour, find the probability that 2 or less sales are made in
   the next hour.
   Solution: (a) Let N (t) be the number of sales made during t hours. N (t) is a Poisson process
   with rate 20 · 0.3 = 6.
                                          E[N (8)] = 48.

   (b)
                                            9
                                            X e−6 6n
                    P (N (1) ≥ 10) = 1 −                = 1 − 0.9161 = 0.0839.
                                            n=0
                                                   n!

   (c) Let Z1 be the time until the first sale. Z1 is exponentially distributed with mean 1/6.
                                                     1
                                             E[Z1 ] = .
                                                     6
   The expected time of the first sale is 10:10am.
   (d)
                                                                   2
                                                                   X e−6 6n
            P(N (2) − N (1) ≤ 2|N (1) = 6) = P(N (1) ≤ 2) =                    = 0.06197.
                                                                   n=0
                                                                          n!




                                               5

<!-- page 6 -->
2. The claim department of an insurance company receives envelops with claims for insurance
   coverage at a Poisson rate of λ = 50 envelopes per week. For any period of time, the number
   of envelopes and the number of claims in the envelopes are independent. The numbers of
   claims in the envelopes, Xi , i = 1, 2, · · · , are i.i.d. distributed with

       P (X = 1) = 0.2,     P (X = 2) = 0.25,      P (X = 3) = 0.4,       P (X = 4) = 0.15.

   Using the normal approximation, calculate the 90th percentile of the aggregate number of
   claims received in 13 weeks.
   Solution: Let N (t) be the number of envelopes received until time t. N (t) is a Poisson
   process with rate λ = 50 per week. Let {Xi } be the number of claims received in each
   envelope. The total number of claims received in 13 weeks is
                                                N (13)
                                                X
                                         Y =             Xi .
                                                 i=1

   We then have
                                  E[X] = 2.5,      E[X 2 ] = 7.2.
                         E[Y ] = E[N (13)]E[X] = 13 · 50 · 7.2 = 1625.
                              V ar(Y ) = E[N (13)]E[X 2 ] = 4680.
   Let Q90 be the 90-th percentile of the number of claims received in 13 weeks. Then we have
                                                    Q90 − 1625
                          P(X ≤ Q90 ) = P(Z ≤         √        ) = 0.9,
                                                        4680
                                      √
   where Z ∼ N (0, 1). Q90 = 1625 +    4680 · 1.282 = 1712.70.




                                            6

<!-- page 7 -->
3. Consider a permanent disability model with three states: healthy (0), permanent disabled (1),
   and death (2). The generator matrix for an individual within a year is
                                                             
                                         −0.06 0.05 0.01
                                G= 0            −0.1 0.1  .
                                            0       0     0

   Transition probabilities are estimated using Kolmogorov’s forward equations solved numeri-
   cally with Euler’s forward method. A step of h = 0.1 is used.
   Estimate the probability that a life healthy currently is disabled at time 0.2, p01 (0.2), and the
   probability that a life healthy currently is healthy at time 0.2, p00 (0.2).
   Solution: In the first period of length 0.1,

                                p00 (0.1) − p00 (0)
                                                    = − p00 (0) (0.06) .
                                        0.1

                                 p00 (0.1) = 1 − 0.006 = 0.994.

                           p01 (0.1) − p01 (0)
                                               = p00 (0)0.05 − p01 (0)0.1.
                                   0.1
                                   p01 (0.1) = 0.1 · 0.05 = 0.005.

   In the second period of length 0.1,

                       p01 (0.2) − p01 (0.1)
                                             = p00 (0.1) · 0.05 − p01 (0.1) · 0.1.
                                0.1

            p01 (0.2) = 0.005 + 0.1 × 0.994 × 0.05 − 0.1 · 0.005 · 0.1 = 0.00992.


                            p00 (0.2) − p00 (0.1)
                                                  = −p00 (0.1) · 0.06.
                                     0.1
   p00 (0.2) = 0.9942 = 0.988036.




                                                  7

<!-- page 8 -->
4. Consider the continuous-time Markov chain with state space 1, 2, 3, 4 and infinitesimal gen-
   erator
                                                 
                                        −2 1 1  0
                                       0 −1 1  0 
                                                 .
                                       1  1 −3 1 
                                         0 0 1 −1
   (a) Find the stationary distribution π.
   (b) Suppose the chain starts in state 1. What is the expected amount of time until it changes
   state for the first time?
   (c) Again assume the chain starts in state 1. What is the expected amount of time until the
   chain is in state 4?
   Solution: (a) Solving the balance equation
                                                 
                                  −2 1       1 0
                                0 −1 1        0 
                      πG = π                      = 0, with π1 = 1,
                                1      1 −3 1 
                                   0    0    1 −1

   we obtain the stationary distribution π = 81 (1, 3, 2, 2) .
   (b) The sojourn time of state 1 U1 ∼ Exp(2). Hence, given X(0) = 1, the expected amount
   of time until it changes state for the first time is given by
                                                       1
                                              E [U1 ] = .
                                                       2

   (c) Since                                    −1   
                                         2 −1 −1        4
                        (−G−4 )−1 1 =  0  1 −1  1 =  4  ,
                                        −1 −1 3         3
   given the chain starts in state 1, the expected amount of time until the chain is in state 4 is
   equal to 4.




                                                8

<!-- page 9 -->
5. (Graduate Students Only) In a given city, a taxi driver can only pick up/drop off customers
   at two different locations. Once a drop off is made, the taxi driver must wait at the same
   location for new customers. You are provided with the following information:

      • At the two locations, the waiting time to pick up a new customer is exponentially
        distributed with a mean of 5 minutes. The customer will be dropped off at the other
        location with probability 1.
      • The travel times between the two locations are exponentially distributed with a mean of
        10 minutes (from location 1 to 2) and a mean of 11 minutes (from location 2 to 1).

   Design a continuous-time Markov chain (CTMC) capable to identify the limiting probability
   that the taxi driver is waiting for a customer at location 1. Identify clearly all states of this
   CTMC.
   Solution: We consider a CTMC with the following state space:

                               state 0:   Waiting at location 1
                               state 1:   Waiting at location 2
                               state 2:   Driving from location 1 to 2
                               state 3:   Driving from location 2 to 1

   It follows that the infinitesimal generator of this CTMC is
                                                                 
                                        −0.2      0     0.2    0
                                       0       −0.2     0    0.2 
                                                                  .
                                 Q=   0        0.1 −0.1 0 
                                          1                     1
                                          11
                                                  0      0   − 11

   The CTMC is irreducible with a finite state space S = {0, 1, 2, 3}, so the stationary distribu-
   tion exists and is unique. It follows that
                                                      1
                                           0.2π0 =      π3
                                                     11
                                           0.2π1   = 0.1π2
                                           0.2π0   = 0.1π2

   together with
                                        π0 + π1 + π2 + π3 = 1.
   Therefore,
                                              1
                          π2 = 2π0 ;      π1 = π2 = π0 ;         π3 = 2.2π0 .
                                              2
   It follows that
                                               1
                                       π0 = π1 =  = 0.16129.
                                              6.2
   The fact that the CTMC is irreducible with a finite state space S = {0, 1, 2, 3} also implies
   that
                                  lim pi0 (t) = π0 = 0.16129.
                                    t→∞



                                               9

<!-- page 10 -->
6. (Graduate Students Only) Consider a two-server system in which a customer is served first
   by server 1, then by server 2, and then departs. The service times at server i are exponential
   rv’s with intensity λi for i = 1, 2. When you arrive, you find server 1 free and two customers
   at server 2 - customer A in service and customer B waiting in line.
   (a) Find PA , the probability that A is still in service when you move over to server 2.
   Solution: Let Xi be the service time at server i. Then,

                           PA = P (X1 < X2 )
                                Z ∞
                              =      λ1 e−λ1 t P (X1 < X2 |X1 = t ) dt
                                Z0 ∞
                              =      λ1 e−λ1 t e−λ2 t dt
                                      0
                                        λ1
                                 =           .
                                     λ1 + λ2

   (b) Find PB , the probability that B is still in the system when you move over to server 2.
   Solution: We have

                            PB = PA + (1 − PA ) PA
                                                          
                                    λ1                 λ1      λ1
                               =         + 1−
                                 λ1 + λ2            λ1 + λ2 λ1 + λ2
                                    λ1          λ1 λ2
                               =         +
                                 λ1 + λ2 (λ1 + λ2 )2
                                 λ1 (λ1 + 2λ2 )
                               =
                                  (λ1 + λ2 )2

   (c) Find E[T ] where T is the time that you spend in the system.
   Solution: The expected time in the queue can be written as

     E = E(T |A and B are both in the queue when you move to server 2)PA
         +E(T |only B is in the queue when you move to server 2)(PB − PA )
         +E(T |A and B are both not in the queue when you move to server 2)(1 − PB ).

   Therefore,

                                                                                   λ22
                                                                                                 
                  λ1            1   3              λ1 λ2          1   2                       1   1
      E [T ] =                    +           +                     +         +                 +
               λ1 + λ2          λ1 λ2           (λ1 + λ2 )2       λ1 λ2         λ1 + λ2       λ1 λ2
                  λ21 (2λ1 + 3λ2 ) + (λ1 + λ2 )3
                =                                .
                         λ1 λ2 (λ1 + λ2 )2




                                                  10

<!-- page 11 -->
7. Consider a birth-death process with birth rates
                                        
                                           n + 1, n = 0, 1, 2
                                  λn =
                                           n,      n≥3
   and death rates
                                    βn = 2n,      n = 0, 1, 2, . . .
   Determine the proportion of time that the population is greater or equal to 3.

   Solution: The generator is given by
                               −1 1
                                                              
                            2 −4 2                            
                                                              
                           
                                    4 −7 3                    
                                                               
                     G=
                                      6  −9   3               
                                          .. ..   ..
                                                               
                           
                                           .   .    .         
                                                               
                                              2n −3n n
                                                              
                                                              
                                                  ..   .. ..
                                                     .   .   .

   From πG = 0 and π1 = 1, the stationary distribution solves

                                             −π0 + 2π1           =     0,
                                         π0 − 4π1 + 4π2          =     0,
                                        2π1 − 7π2 + 6π3          =     0
                                        3π2 − 9π3 + 8π4          =     0
                      nπn − 3(n + 1)πn+1 + 2(n + 2)πn+2          =     0,       n ≥ 3.

   To solve for πn+2 , we can add all the previous equations and yields

                               −(n + 1)πn+1 + 2(n + 2)πn+2 = 0,

   or equivalently,
                                                n+1
                                      πn+2 =            πn+1 .
                                               2(n + 2)
   This implies
                                        1
                               π1 =       π0 ,
                                        2
                                                2
                                        1         1
                               π2 =       π1 =       π0
                                        2         2
                                                3
                                        1         1
                               π3 =       π2 =       π0
                                        2         2
                                                   4
                                        3       3 1
                               π4 =       π3 =         π0
                                        8       4 2
                                                n+2
                                           3     1
                            πn+2 =                     π0 ,            n ≥ 3.
                                        n+2 2

                                             11

<!-- page 12 -->
             P∞
Given that    n=0 πn = 1, we have

                                                            ∞  1 n
                                                    2           !
                                          1         1      X
                              1 = π0    1+ +            +3     2
                                                                     .
                                          2         2      n=3
                                                               n

Note that
             ∞      1 n       ∞     1 n
                                                        
             X
                    2
                              X
                                    2      1 1           1    5         5
                          =               − − = − ln 1 −     − = ln 2 −
              n=3
                    n         n=1
                                    n      2 8           2    8         8

It follows that
                                                   1
                                       π0 =               = 0.51166.
                                              3 ln 2 − 81
The proportion of time that the population is greater or equal to 3 is given by
                                                      
                                                 1 1
                1 − π0 − π 1 − π2 = 1 − 1 + +             0.51166 = 0.1046.
                                                 2 4




                                                 12

<!-- page 13 -->
8. A small barbershop operated by a single barber has room for at most two customers. Potential
   customers arrive at a Poisson rate of three per hour, and the successive service times are
   independent exponential random variables with mean 0.25 hours.
   (a) What is the average number of customers in the shop in the long run?
   (b) What is the proportion of customers that enter the shop, i.e., when the barbershop has 0
   or 1 customers?
   (c) If the barber could work twice as fast, how much more business would he do?
   Solution: (a) This CTMC has state space {0, 1, 2}. Since the successive service times are
   independent exponential random variables with mean 1/4 per hour, we know that q10 =
   q21 = 4. Hence the generator of the CTMC is given by
                                                       
                                          −3 3       0
                                   G =  4 −7 3  .
                                          0     4 −4
   By solving πG = 0 and π1 = 1, the stationary distribution is given by
                                              1
                                       π=       (16, 12, 9) .
                                             37
   Thus, the average number of customers in the shop in the long run equals
                                                        30
                                           π1 + 2π2 =      .
                                                        37



   (b) The proportion of potential costumers entering the shop is equal to the fraction of time
   that the shop has 0 or 1 costumers, which is equal to
                                                        28
                                           π0 + π 1 =      .
                                                        37

   (c) If the barber works twice as fast, the generator becomes
                                                            
                                              −3 3         0
                                    G =  8 −11 3  .
                                               0     8 −8
   By solving πG = 0 and π1 = 1, the stationary distribution is given by
                                              1
                                       π=       (64, 24, 9) .
                                             97
   Then proportion of costumers entering the shop is equal to
                                                        88
                                           π0 + π 1 =      .
                                                        97
                                      88
   Thus, the business has a growth of 97
                                      28 − 1 = 19.88%.
                                      37




                                              13

