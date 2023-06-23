------------------------------- MODULE Target -------------------------------

INSTANCE Naturals

----

\* An empty constant
CONSTANT Nil

\* The set of possible master nodes
CONSTANT Node

----

\* The target state
VARIABLE target

\* A record of connections between nodes and the target
VARIABLE conns

TypeOK ==
   /\ target.id \in Nat
   /\ \A p \in DOMAIN target.values :
         /\ target.values[p].index \in Nat
         /\ target.values[p].value \in STRING
   /\ target.running \in BOOLEAN 
   /\ \A n \in DOMAIN conns : 
         /\ n \in Node
         /\ conns[n].id \in Nat
         /\ conns[n].connected \in BOOLEAN 

----

(*
This section specifies the behavior of configuration targets.
*)

Start ==
   /\ ~target.running
   /\ target' = [target EXCEPT !.id      = target.id + 1,
                               !.running = TRUE]
   /\ UNCHANGED <<conns>>

Stop ==
   /\ target.running
   /\ target' = [target EXCEPT !.running = FALSE,
                               !.values  = [p \in {} |-> [index |-> 0, value |-> Nil]]]
   /\ conns' = [n \in Node |-> [conns[n] EXCEPT !.connected = FALSE]]

----

(*
This section specifies the behavior of connections to the target.
*)

Connect(n) ==
   /\ ~conns[n].connected
   /\ target.running
   /\ conns' = [conns EXCEPT ![n].id        = conns[n].id + 1,
                             ![n].connected = TRUE]
   /\ UNCHANGED <<target>>

Disconnect(n) ==
   /\ conns[n].connected
   /\ conns' = [conns EXCEPT ![n].connected = FALSE]
   /\ UNCHANGED <<target>>

=============================================================================
