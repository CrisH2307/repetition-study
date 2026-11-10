<!-- page 1 -->
Notes on exponential races
An exponential random variable with rate a has the density a e^(-a t) for t at least zero.
Its tail is P(T > t) = e^(-a t), so waiting longer makes the event exponentially less likely.
Suppose X has rate a and Y has rate b, and the two are independent.
Then the chance that X finishes before Y is a / (a + b).
To see this, condition on X = t, and note that Y must then exceed t, which has probability e^(-b t).
Integrating a e^(-a t) times e^(-b t) over t from zero to infinity gives a / (a + b).
The smaller of the two times, min(X, Y), is again exponential, with rate a + b.
This follows because min(X, Y) > t happens exactly when both exceed t, and the tails multiply.
<!-- page 2 -->
Notes on superposition of Poisson processes
A Poisson process with rate a has independent exponential gaps between arrivals, each with rate a.
Merging two independent Poisson processes with rates a and b gives one Poisson process with rate a + b.
Each arrival in the merged stream came from the first process with probability a / (a + b), independently of the others.
So if n arrivals happen in the merged stream, the number from the first process is binomial with n trials and success chance a / (a + b).
The count of arrivals in a window of length s is Poisson with mean (a + b) s.
The next arrival from either process is an exponential race, so it comes after an Exp(a + b) wait.
