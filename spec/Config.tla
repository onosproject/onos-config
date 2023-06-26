This module provides the model values and safety and liveness properties for
the model of µONOS Config controllers.

------------------------------- MODULE Config -------------------------------

INSTANCE Naturals

INSTANCE FiniteSets

INSTANCE Sequences

INSTANCE TLC

----

GenerateTestCases == TRUE

Nil == "<nil>"

Change == "Change"
Rollback == "Rollback"

Commit == "Commit"
Apply == "Apply"

Pending == "Pending"
InProgress == "InProgress"
Complete == "Complete"
Aborted == "Aborted"
Canceled == "Canceled"
Failed == "Failed"

Node == {"node1"}

NumTransactions == 3
NumTerms == 1
NumConns == 1
NumStarts == 1

Path == {"path1"}
Value == {"value1", "value2"}

----

\* A transaction log.
VARIABLE transactions

\* A record of per-target configurations
VARIABLE configuration

\* A record of target masterships
VARIABLE mastership

\* A record of node connections to the target
VARIABLE conns

\* The target state
VARIABLE target

\* A sequence of state changes used for model checking.
VARIABLE history

vars == <<transactions, configuration, mastership, conns, target, history>>

----

(*
µONOS Config coordinates configuration changes using a collection of four 
controllers, each of which is responsible for managing one type of data. The 
four controllers are specified in a separate module for each, and they're
imported here for use in the model.
*)

LOCAL Transaction == INSTANCE Transaction

LOCAL Configuration == INSTANCE Configuration

LOCAL Mastership == INSTANCE Mastership

LOCAL Target == INSTANCE Target

----

(*
This section defines the state changes that can occur across all
the µONOS Config controllers.
*)

AppendChange(i) ==
   /\ Transaction!AppendChange(i)

RollbackChange(i) ==
   /\ Transaction!RollbackChange(i)

ReconcileTransaction(n, i) ==
   /\ i \in DOMAIN transactions
   /\ \/ /\ Transaction!ReconcileTransaction(n, i)
         /\ GenerateTestCases => Transaction!Test!Log([node |-> n, index |-> i])
      \/ /\ GenerateTestCases
         /\ ~ENABLED Transaction!ReconcileTransaction(n, i)
         /\ UNCHANGED vars
         /\ Transaction!Test!Log([node |-> n, index |-> i])

ReconcileConfiguration(n) ==
   \/ /\ Configuration!ReconcileConfiguration(n)
      /\ UNCHANGED <<transactions, history>>
      /\ GenerateTestCases => Configuration!Test!Log([node |-> n])
   \/ /\ GenerateTestCases
      /\ ~ENABLED Configuration!ReconcileConfiguration(n)
      /\ UNCHANGED vars
      /\ Configuration!Test!Log([node |-> n])

ReconcileMastership(n) ==
   \/ /\ Mastership!ReconcileMastership(n)
      /\ UNCHANGED <<transactions, configuration, target, history>>
      /\ GenerateTestCases => Mastership!Test!Log([node |-> n])
   \/ /\ GenerateTestCases
      /\ ~ENABLED Mastership!ReconcileMastership(n)
      /\ UNCHANGED vars
      /\ Mastership!Test!Log([node |-> n])

ConnectNode(n) ==
   /\ Target!Connect(n)
   /\ UNCHANGED <<transactions, configuration, mastership, history>>

DisconnectNode(n) ==
   /\ Target!Disconnect(n)
   /\ UNCHANGED <<transactions, configuration, mastership, history>>

StartTarget ==
   /\ Target!Start
   /\ UNCHANGED <<transactions, configuration, mastership, history>>

StopTarget ==
   /\ Target!Stop
   /\ UNCHANGED <<transactions, configuration, mastership, history>>

----

(*
Formal specification, constraints, and theorems.
*)

Init ==
   /\ transactions = [
         i \in {} |-> [
            phase    |-> Nil,
            values |-> [
               p \in {} |-> Nil],
            change   |-> [
               commit |-> Nil,
               apply  |-> Nil],
            rollback |-> [
               commit |-> Nil,
               apply  |-> Nil]]]
   /\ configuration = [
         state  |-> Pending,
         term   |-> 0,
         committed |-> [
            index    |-> 0,
            change   |-> 0,
            target   |-> 0,
            ordinal  |-> 0,
            revision |-> 0,
            values   |-> [
               p \in {} |-> Nil]],
         applied |-> [
            index    |-> 0,
            target   |-> 0,
            ordinal  |-> 0,
            revision |-> 0,
            values   |-> [
               p \in {} |-> Nil]]]
   /\ target = [
         id      |-> 1,
         running |-> TRUE,
         values  |-> [
            p \in {} |-> [
               index |-> 0, 
               value |-> Nil]]]
   /\ mastership = [
         master |-> CHOOSE n \in Node : TRUE, 
         term   |-> 1,
         conn   |-> 1]
   /\ conns = [
         n \in Node |-> [
            id        |-> 1,
            connected |-> TRUE]]
   /\ history = <<>>

Next ==
   \/ \E i \in 1..NumTransactions :
         \/ AppendChange(i)
         \/ RollbackChange(i)
   \/ \E n \in Node, i \in 1..NumTransactions :
         ReconcileTransaction(n, i)
   \/ \E n \in Node :
         ReconcileConfiguration(n)
   \/ \E n \in Node :
         ReconcileMastership(n)
   \/ \E n \in Node :
         \/ ConnectNode(n)
         \/ DisconnectNode(n)
   \/ StartTarget
   \/ StopTarget

Spec ==
   /\ Init
   /\ [][Next]_vars
   /\ \A i \in 1..NumTransactions :
         WF_<<transactions>>(Transaction!RollbackChange(i))
   /\ \A n \in Node, i \in 1..NumTransactions :
         WF_<<transactions, configuration, mastership, conns, target, history>>(Transaction!ReconcileTransaction(n, i))
   /\ \A n \in Node :
         WF_<<configuration, mastership, conns, target>>(Configuration!ReconcileConfiguration(n))
   /\ \A n \in Node :
         WF_<<mastership, conns>>(Mastership!ReconcileMastership(n))
   /\ \A n \in Node :
         WF_<<conns, target>>(Target!Connect(n) \/ Target!Disconnect(n))
   /\ WF_<<conns, target>>(Target!Start \/ Target!Stop)

----

(*
This section contains state constraints used for model checking.
*)

LimitTerms == 
   \/ mastership.term < NumTerms
   \/ /\ mastership.term = NumTerms
      /\ mastership.master # Nil

LimitConns ==
   \A n \in DOMAIN conns :
      \/ conns[n].id < NumConns
      \/ /\ conns[n].id = NumConns 
         /\ conns[n].connected

LimitStarts ==
   \/ target.id < 2
   \/ /\ target.id = 2
      /\ target.running

TypeOK ==
   /\ Transaction!TypeOK
   /\ Configuration!TypeOK
   /\ Mastership!TypeOK

----

(*
This section contains invariants, action properties, and liveness properties
used the model to verify the spec preserves order and consistency guarantees,
always eventually terminates, and is deadlock free.
*)

StatusCommitted(i) ==
   /\ Len(history) = Len(history')
   /\ \/ /\ transactions'[i].change.commit \notin {Pending, Canceled}
         /\ transactions[i].change.commit # transactions'[i].change.commit
      \/ /\ transactions'[i].rollback.commit \notin {Pending, Canceled}
         /\ transactions[i].rollback.commit # transactions'[i].rollback.commit

StatusApplied(i) ==
   /\ Len(history) = Len(history')
   /\ \/ /\ transactions'[i].change.apply \notin {Pending, Canceled, Aborted}
         /\ transactions[i].change.apply # transactions'[i].change.apply
      \/ /\ transactions'[i].rollback.apply \notin {Pending, Canceled, Aborted}
         /\ transactions[i].rollback.apply # transactions'[i].rollback.apply

ValidStatus(t, i, j) ==
   /\ j \in DOMAIN history
   /\ history[j].index = i
   /\ \/ /\ history[j].phase = Change
         /\ history[j].event = Commit
         /\ t[i].change.commit = history[j].status
      \/ /\ history[j].phase = Change
         /\ history[j].event = Apply
         /\ t[i].change.apply = history[j].status
      \/ /\ history[j].phase = Rollback
         /\ history[j].event = Commit
         /\ t[i].rollback.commit = history[j].status
      \/ /\ history[j].phase = Rollback
         /\ history[j].event = Apply
         /\ t[i].rollback.apply = history[j].status

ValidCommit(t, i) ==
   LET j == CHOOSE j \in DOMAIN history :
               /\ history[j].event = Commit
               /\ ~\E k \in DOMAIN history :
                     /\ history[k].event = Commit
                     /\ k > j
   IN ValidStatus(t, i, j)

ValidApply(t, i) ==
   LET j == CHOOSE j \in DOMAIN history :
               /\ history[j].event = Apply
               /\ ~\E k \in DOMAIN history :
                     /\ history[k].event = Apply
                     /\ k > j
   IN ValidStatus(t, i, j)

AtomicStatusChange ==
   \A i \in 1..NumTransactions :
      /\ i \in DOMAIN transactions =>
            /\ StatusCommitted(i) => ValidCommit(transactions', i)
            /\ StatusApplied(i) => ValidApply(transactions', i)

Transition == [][AtomicStatusChange]_<<transactions, history>>

LOCAL IsOrderedChange(p, i) ==
   /\ history[i].phase = Change
   /\ history[i].event = p
   /\ history[i].status = Complete
   /\ ~\E j \in DOMAIN history :
         /\ j < i
         /\ history[j].phase = Change
         /\ history[j].event = p
         /\ history[j].status = Complete
         /\ history[j].index >= history[i].index

LOCAL IsOrderedRollback(p, i) ==
   /\ history[i].phase = Rollback
   /\ history[i].event = p
   /\ history[i].status = Complete
   /\ \E j \in DOMAIN history :
         /\ j < i
         /\ history[j].phase = Change
         /\ history[j].status = Complete
         /\ history[j].index = history[i].index
   /\ ~\E j \in DOMAIN history :
         /\ j < i
         /\ history[j].phase = Change
         /\ history[j].event = p
         /\ history[j].status = Complete
         /\ history[j].index > history[i].index
         /\ ~\E k \in DOMAIN history :
               /\ k > j
               /\ k < i
               /\ history[k].phase = Rollback
               /\ history[k].event = p
               /\ history[j].status = Complete
               /\ history[k].index = history[j].index

\* The Order invariant checks the recorded 'history' to ensure that changes and rollbacks
\* are being committed and applied in consistent sequential order. Additionally, it enforces
\* the invariant that if a rollback fails the apply phase, it and all subsequent pending 
\* transactions must be aborted and rolled back before a new transaction can be applied.
Order ==
   /\ \A i \in DOMAIN history :
         history[i].status = Complete => 
            \/ IsOrderedChange(Commit, i)
            \/ IsOrderedChange(Apply, i)
            \/ IsOrderedRollback(Commit, i)
            \/ IsOrderedRollback(Apply, i)
   /\ \A i \in DOMAIN transactions :
         /\ transactions[i].change.apply = Failed
         /\ transactions[i].rollback.apply # Complete
         => ~\E j \in DOMAIN transactions : 
               /\ j > i
               /\ transactions[i].change.apply \in {InProgress, Complete}

LOCAL IsChangeCommitted(i) ==
   /\ configuration.committed.revision = i

LOCAL IsChangeApplied(i) ==
   /\ configuration.applied.revision = i

\* The Consistency invariant verifies that the changes and rollbacks are properly
\* propagated to the configuration when committed and to targets once applied.
\* Additionally, it checks that target configuration is restored following restarts.
Consistency ==
   /\ \A i \in DOMAIN transactions :
         /\ IsChangeCommitted(i)
         /\ ~\E j \in DOMAIN transactions :
               /\ j > i
               /\ IsChangeCommitted(j)
         => \A p \in DOMAIN transactions[i].change.values :
               /\ configuration.committed.values[p] = transactions[i].change.values[p]
   /\ \A i \in DOMAIN transactions :
         /\ IsChangeApplied(i)
         /\ ~\E j \in DOMAIN transactions :
               /\ j > i
               /\ IsChangeApplied(j)
         => \A p \in DOMAIN transactions[i].change.values :
               /\ configuration.applied.values[p] = transactions[i].change.values[p]
               /\ /\ target.running
                  /\ configuration.applied.target = target.id
                  /\ configuration.state = Complete
                  => target.values[p] = transactions[i].change.values[p]

\* The Safety property holds with both Order and Consistency.
Safety == [](Order /\ Consistency)

THEOREM Spec => Safety

LOCAL IsChanging(i) ==
   /\ i \in DOMAIN transactions
   /\ transactions[i].phase = Change

LOCAL IsChanged(i) ==
   /\ i \in DOMAIN transactions
   /\ transactions[i].change.commit \in {Complete, Failed}
   /\ transactions[i].change.apply \in {Complete, Aborted, Failed}

LOCAL IsRollingBack(i) ==
   /\ i \in DOMAIN transactions
   /\ transactions[i].phase = Rollback

LOCAL IsRolledBack(i) ==
   /\ i \in DOMAIN transactions
   /\ transactions[i].rollback.commit \in {Complete, Failed}
   /\ transactions[i].rollback.apply \in {Complete, Aborted, Failed}

Terminates(i) ==
   /\ IsChanging(i) ~> IsChanged(i)
   /\ IsRollingBack(i) ~> IsRolledBack(i)

\* The Termination property is a liveness check to verify no deadlocks or livelocks
\* exist in the system. So long as the system can make progress, every change and
\* every rollback is guaranteed to have a path to complete both phases at all times.
Termination ==
   \A i \in 1..NumTransactions : Terminates(i)

\* The Liveness property holds when Termination holds.
Liveness == Termination

THEOREM Spec => Liveness

=============================================================================

Copyright 2023 Intel Corporation
