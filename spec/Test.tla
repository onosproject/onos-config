------------------------------- MODULE Test -------------------------------

INSTANCE Naturals

INSTANCE Sequences

LOCAL INSTANCE IOUtils

LOCAL INSTANCE Json

CONSTANT File

CONSTANT State

CONSTANT Transitions

FormatOpts == 
   [format      |-> "TXT",
    charset     |-> "UTF-8",
    openOptions |-> <<"WRITE", "CREATE", "APPEND">>]

Delete ==
   /\ IOExec(<<"rm", "-f", File>>).exitValue = 0

Log(context) ==
   LET record == [context |-> context, state |-> State, transitions |-> Transitions]
   IN Serialize(ToJsonObject(record) \o "\n", File, FormatOpts).exitValue = 0

=============================================================================
