------------------------------- MODULE Mastership -------------------------------

INSTANCE Naturals

INSTANCE FiniteSets

INSTANCE Sequences

INSTANCE TLC

----

\* An empty constant
CONSTANT Nil

\* The set of possible master nodes
CONSTANT Node

----

\* Variables defined by other modules.
VARIABLES 
   conns

\* A record of target masterships
VARIABLE mastership

TypeOK ==
   /\ mastership.term \in Nat
   /\ mastership.master # Nil => mastership.master \in Node
   /\ mastership.conn \in Nat

LOCAL State == [
   mastership |-> mastership,
   conns      |-> conns]

LOCAL Transitions ==
   IF mastership' # mastership THEN [mastership |-> mastership'] ELSE <<>>

Test == INSTANCE Test WITH 
   File <- "Mastership.log"

----

(*
This section models mastership for the configuration service.

Mastership is used primarily to track the lifecycle of individual
configuration targets and react to state changes on the southbound.
Each target is assigned a master from the Node set, and masters
can be unset when the target disconnects.
*)

ReconcileMastership(n) ==
   /\ \/ /\ conns[n].connected
         /\ mastership.master = Nil
         /\ mastership' = [
               master |-> n, 
               term   |-> mastership.term + 1,
               conn   |-> conns[n].id]
      \/ /\ \/ ~conns[n].connected
            \/ conns[n].id # mastership.conn
         /\ mastership.master = n
         /\ mastership' = [mastership EXCEPT !.master = Nil]
   /\ UNCHANGED <<conns>>

=============================================================================
