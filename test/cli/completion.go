// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"bytes"
	"errors"
	"github.com/spf13/cobra"
	"io"
	"os"
)

const bashCompletion = `
__onit_get_clusters() {
    local onit_output out
    if onit_output=$(onit get clusters --no-headers 2>/dev/null); then
        out=($(echo "${onit_output}" | awk '{print $1}'))
        COMPREPLY=( $( compgen -W "${out[*]}" -- "$cur" ) )
    fi
}
__onit_get_simulators() {
    local onit_output
    if onit_output=$(onit get simulators 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${onit_output[*]}" -- "$cur" ) )
    fi
}
__onit_get_nodes() {
    local onit_output out
    if onit_output=$(onit get nodes --no-headers 2>/dev/null); then
        out=($(echo "${onit_output}" | awk '{print $1}'))
        COMPREPLY=( $( compgen -W "${out[*]}" -- "$cur" ) )
    fi
}
__onit_custom_func() {
    case ${last_command} in
        onit_set_cluster | onit_delete_cluster)
            __onit_get_clusters
            return
            ;;
        onit_delete_simulator)
            __onit_get_simulators
            return
            ;;
        onit_get_logs | onit_fetch_logs | onit_debug)
            __onit_get_nodes
            return
            ;;
        *)
            ;;
    esac
}
`

func getCompletionCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "completion <shell>",
		Short:     "Generated bash or zsh auto-completion script",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh"},
		Run:       runCompletionCommand,
	}
}

func runCompletionCommand(cmd *cobra.Command, args []string) {
	if args[0] == "bash" {
		if err := runCompletionBash(os.Stdout, cmd.Parent()); err != nil {
			exitError(err)
		}
	} else if args[0] == "zsh" {
		if err := runCompletionZsh(os.Stdout, cmd.Parent()); err != nil {
			exitError(err)
		}

	} else {
		exitError(errors.New("unsupported shell type "+args[0]))
	}
}

func runCompletionBash(out io.Writer, cmd *cobra.Command) error {
	return cmd.GenBashCompletion(out)
}

func runCompletionZsh(out io.Writer, cmd *cobra.Command) error {
	header := "#compdef onit\n"

	out.Write([]byte(header))

	init := `
__onit_bash_source() {
	alias shopt=':'
	alias _expand=_bash_expand
	alias _complete=_bash_comp
	emulate -L sh
	setopt kshglob noshglob braceexpand
	source "$@"
}
__onit_type() {
	# -t is not supported by zsh
	if [ "$1" == "-t" ]; then
		shift
		# fake Bash 4 to disable "complete -o nospace". Instead
		# "compopt +-o nospace" is used in the code to toggle trailing
		# spaces. We don't support that, but leave trailing spaces on
		# all the time
		if [ "$1" = "__onit_compopt" ]; then
			echo builtin
			return 0
		fi
	fi
	type "$@"
}
__onit_compgen() {
	local completions w
	completions=( $(compgen "$@") ) || return $?
	# filter by given word as prefix
	while [[ "$1" = -* && "$1" != -- ]]; do
		shift
		shift
	done
	if [[ "$1" == -- ]]; then
		shift
	fi
	for w in "${completions[@]}"; do
		if [[ "${w}" = "$1"* ]]; then
			echo "${w}"
		fi
	done
}
__onit_compopt() {
	true # don't do anything. Not supported by bashcompinit in zsh
}
__onit_ltrim_colon_completions()
{
	if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
		# Remove colon-word prefix from COMPREPLY items
		local colon_word=${1%${1##*:}}
		local i=${#COMPREPLY[*]}
		while [[ $((--i)) -ge 0 ]]; do
			COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
		done
	fi
}
__onit_get_comp_words_by_ref() {
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[${COMP_CWORD}-1]}"
	words=("${COMP_WORDS[@]}")
	cword=("${COMP_CWORD[@]}")
}
__onit_filedir() {
	local RET OLD_IFS w qw
	__debug "_filedir $@ cur=$cur"
	if [[ "$1" = \~* ]]; then
		# somehow does not work. Maybe, zsh does not call this at all
		eval echo "$1"
		return 0
	fi
	OLD_IFS="$IFS"
	IFS=$'\n'
	if [ "$1" = "-d" ]; then
		shift
		RET=( $(compgen -d) )
	else
		RET=( $(compgen -f) )
	fi
	IFS="$OLD_IFS"
	IFS="," __debug "RET=${RET[@]} len=${#RET[@]}"
	for w in ${RET[@]}; do
		if [[ ! "${w}" = "${cur}"* ]]; then
			continue
		fi
		if eval "[[ \"\${w}\" = *.$1 || -d \"\${w}\" ]]"; then
			qw="$(__onit_quote "${w}")"
			if [ -d "${w}" ]; then
				COMPREPLY+=("${qw}/")
			else
				COMPREPLY+=("${qw}")
			fi
		fi
	done
}
__onit_quote() {
    if [[ $1 == \'* || $1 == \"* ]]; then
        # Leave out first character
        printf %q "${1:1}"
    else
    	printf %q "$1"
    fi
}
autoload -U +X bashcompinit && bashcompinit
# use word boundary patterns for BSD or GNU sed
LWORD='[[:<:]]'
RWORD='[[:>:]]'
if sed --help 2>&1 | grep -q GNU; then
	LWORD='\<'
	RWORD='\>'
fi
__onit_convert_bash_to_zsh() {
	sed \
	-e 's/declare -F/whence -w/' \
	-e 's/_get_comp_words_by_ref "\$@"/_get_comp_words_by_ref "\$*"/' \
	-e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
	-e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
	-e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
	-e "s/${LWORD}_filedir${RWORD}/__onit_filedir/g" \
	-e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__onit_get_comp_words_by_ref/g" \
	-e "s/${LWORD}__ltrim_colon_completions${RWORD}/__onit_ltrim_colon_completions/g" \
	-e "s/${LWORD}compgen${RWORD}/__onit_compgen/g" \
	-e "s/${LWORD}compopt${RWORD}/__onit_compopt/g" \
	-e "s/${LWORD}declare${RWORD}/builtin declare/g" \
	-e "s/\\\$(type${RWORD}/\$(__onit_type/g" \
	<<'BASH_COMPLETION_EOF'
`
	out.Write([]byte(init))

	buf := new(bytes.Buffer)
	cmd.GenBashCompletion(buf)
	out.Write(buf.Bytes())

	tail := `
BASH_COMPLETION_EOF
}
__onit_bash_source <(__onit_convert_bash_to_zsh)
_complete onit 2>/dev/null
`
	out.Write([]byte(tail))
	return nil
}
