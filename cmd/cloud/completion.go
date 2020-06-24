// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	completionCmd.AddCommand(bashCmd)
	completionCmd.AddCommand(zshCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generates autocompletion scripts for bash and zsh",
}

var bashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generates the bash autocompletion scripts",
	Long: `To load completion, run

. <(cloud completion bash)

To configure your bash shell to load completions for each session, add the above line to your ~/.bashrc
`,
	Run: func(command *cobra.Command, args []string) {
		rootCmd.GenBashCompletion(os.Stdout)
	},
}

var zshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generates the zsh autocompletion scripts",
	Long: `To load completion, run

. <(cloud completion zsh)

To configure your zsh shell to load completions for each session, add the above line to your ~/.zshrc
`,
	Run: func(command *cobra.Command, args []string) {
		zshInitialization := `
__cloud_bash_source() {
	alias shopt=':'
	alias _expand=_bash_expand
	alias _complete=_bash_comp
	emulate -L sh
	setopt kshglob noshglob braceexpand
	source "$@"
}
__cloud_type() {
	# -t is not supported by zsh
	if [ "$1" == "-t" ]; then
		shift
		# fake Bash 4 to disable "complete -o nospace". Instead
		# "compopt +-o nospace" is used in the code to toggle trailing
		# spaces. We don't support that, but leave trailing spaces on
		# all the time
		if [ "$1" = "__cloud_compopt" ]; then
			echo builtin
			return 0
		fi
	fi
	type "$@"
}
__cloud_compgen() {
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
__cloud_compopt() {
	true # don't do anything. Not supported by bashcompinit in zsh
}
__cloud_declare() {
	if [ "$1" == "-F" ]; then
		whence -w "$@"
	else
		builtin declare "$@"
	fi
}
__cloud_ltrim_colon_completions()
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
__cloud_get_comp_words_by_ref() {
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[${COMP_CWORD}-1]}"
	words=("${COMP_WORDS[@]}")
	cword=("${COMP_CWORD[@]}")
}
__cloud_filedir() {
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
			qw="$(__cloud_quote "${w}")"
			if [ -d "${w}" ]; then
				COMPREPLY+=("${qw}/")
			else
				COMPREPLY+=("${qw}")
			fi
		fi
	done
}
__cloud_quote() {
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
__cloud_convert_bash_to_zsh() {
	sed \
	-e 's/declare -F/whence -w/' \
	-e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
	-e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
	-e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
	-e "s/${LWORD}_filedir${RWORD}/__cloud_filedir/g" \
	-e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__cloud_get_comp_words_by_ref/g" \
	-e "s/${LWORD}__ltrim_colon_completions${RWORD}/__cloud_ltrim_colon_completions/g" \
	-e "s/${LWORD}compgen${RWORD}/__cloud_compgen/g" \
	-e "s/${LWORD}compopt${RWORD}/__cloud_compopt/g" \
	-e "s/${LWORD}declare${RWORD}/__cloud_declare/g" \
	-e "s/\\\$(type${RWORD}/\$(__cloud_type/g" \
	<<'BASH_COMPLETION_EOF'
`

		zshTail := `
BASH_COMPLETION_EOF
}
__cloud_bash_source <(__cloud_convert_bash_to_zsh)
`

		os.Stdout.Write([]byte(zshInitialization))
		rootCmd.GenBashCompletion(os.Stdout)
		os.Stdout.Write([]byte(zshTail))
	},
}
