------------------------------- MODULE Configuration -------------------------------

INSTANCE Naturals

INSTANCE FiniteSets

INSTANCE Sequences

INSTANCE TLC

----

\* An empty constant
CONSTANT Nil

\* Status constants
CONSTANTS
   Pending,
   Complete

Status == 
   {Pending,
    Complete}

----

\* Variables defined by other modules.
VARIABLES 
   mastership,
   conns,
   target

\* A record of per-target configurations
VARIABLE configuration

TypeOK ==
   /\ configuration.state \in Status
   /\ configuration.term \in Nat
   /\ \A p \in DOMAIN configuration.committed.values :
         /\ configuration.committed.index \in Nat
         /\ configuration.committed.revision \in Nat
         /\ configuration.committed.values[p] # Nil =>
               configuration.committed.values[p] \in STRING
   /\ configuration.applied.target \in Nat
   /\ \A p \in DOMAIN configuration.applied.values :
         /\ configuration.applied.index \in Nat
         /\ configuration.applied.revision \in Nat
         /\ configuration.applied.values[p] # Nil =>
               configuration.applied.values[p] \in STRING

LOCAL State == [
   configuration |-> configuration,
   mastership    |-> mastership,
   conns         |-> conns,
   target        |-> target]

LOCAL Transitions ==
   <<>> @@
   (IF configuration' # configuration THEN [configuration |-> configuration'] ELSE <<>>) @@
   (IF target' # target THEN [target |-> target'] ELSE <<>>)

Test == INSTANCE Test WITH 
   File <- "Configuration.log"

----

(*
This section models the Configuration reconciler.
*)

ReconcileConfiguration(n) ==
   /\ \/ /\ configuration.state = Pending
         /\ configuration.term = mastership.term
         /\ mastership.master = n
         /\ conns[n].id = mastership.conn
         /\ conns[n].connected
         /\ target.running
         /\ target' = [target EXCEPT !.values = configuration.applied.values]
         /\ configuration' = [configuration EXCEPT !.state = Complete]
      \/ /\ configuration.term < mastership.term
         /\ configuration' = [configuration EXCEPT !.state = Pending,
                                                   !.term  = mastership.term]
         /\ UNCHANGED <<target>>
   /\ UNCHANGED <<mastership, conns>>

=============================================================================
