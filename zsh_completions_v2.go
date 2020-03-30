package cobra

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

func genZshCompV2(buf *bytes.Buffer, name string, includeDesc bool) {
	compCmd := CompRequestCmd
	if includeDesc {
		compCmd = CompWithDescRequestCmd
	}
	buf.WriteString(fmt.Sprintf("# zsh completion for %-36s -*- shell-script -*-\n", name))
	buf.WriteString(fmt.Sprintf(`
__%[1]s_debug()
{
    local file="$BASH_COMP_DEBUG_FILE"
    if [[ -n ${file} ]]; then
        echo "$*" >> "${file}"
    fi
}

__%[1]s_do_completion()
{
    local lastParam lastChar flagPrefix requestComp out directive compCount comp lastComp
    local -a completions

    __%[1]s_debug "\n========= starting completion logic =========="
    __%[1]s_debug "CURRENT: ${CURRENT}, words[*]: ${words[*]}"

    lastParam=${words[-1]}
    lastChar=${lastParam[-1]}
    __%[1]s_debug "lastParam: ${lastParam}, lastChar: ${lastChar}"

    # For zsh, when completing a flag with an = (e.g., %[1]s -n=<TAB>)
    # completions must be prefixed with the flag
    setopt local_options BASH_REMATCH
    if [[ "${lastParam}" =~ '-.*=' ]]; then
        # We are dealing with a flag with an =
        flagPrefix=${BASH_REMATCH}
    fi

    # Prepare the command to obtain completions
    requestComp="${words[1]} %[2]s ${words[2,-1]}"
    if [ "${lastChar}" = "" ]; then
        # If the last parameter is complete (there is a space following it)
        # We add an extra empty parameter so we can indicate this to the go completion code.
        __%[1]s_debug "Adding extra empty parameter"
        requestComp="${requestComp} \"\""
    fi

    __%[1]s_debug "About to call: eval ${requestComp}"

    # Use eval to handle any environment variables and such
    out=$(eval ${requestComp} 2>/dev/null)
    __%[1]s_debug "completion output: ${out}"

    # Extract the directive integer following a : as the last line
    if [ "${out[-2]}" = : ]; then
        directive=${out[-1]}
        # Remove the directive (that means the last 3 chars as we include the : and the newline)
        out=${out[1,-4]}
    else
        # There is not directive specified.  Leave $out as is.
        __%[1]s_debug "No directive found.  Setting do default"
        directive=0
    fi

    __%[1]s_debug "directive: ${directive}"
    __%[1]s_debug "completions: ${out}"
    __%[1]s_debug "flagPrefix: ${flagPrefix}"

    if [ $((directive & %[3]d)) -ne 0 ]; then
        __%[1]s_debug "Completion received error. Ignoring completions."
    else
        compCount=0
        while IFS='\n' read -r comp; do
            if [ -n "$comp" ]; then
                ((compCount++))
                if [ -n "$flagPrefix" ]; then
                    # We use compadd here so that we can hide the flagPrefix from the list
                    # of choices. We can use compadd because there is no description in this case.
                    __%[1]s_debug "Calling: compadd -p ${flagPrefix} ${comp}"
                    compadd -p ${flagPrefix} ${comp}
                else
                    # If requested, completions are returned with a description.
                    # The description is preceded by a TAB character.
                    # For zsh's _describe, we need to use a : instead of a TAB.
                    # We first need to escape any : as part of the completion itself.
                    comp=${comp//:/\\:}

                    local tab=$(printf '\t')
                    comp=${comp//$tab/:}

                    __%[1]s_debug "Adding completion: ${comp}"
                    completions+=${comp}
                fi
                lastComp=$comp
            fi
        done < <(printf "%%s\n" "${out[@]}")

        if [ ${compCount} -eq 0 ]; then
            if [ $((directive & %[5]d)) -ne 0 ]; then
                __%[1]s_debug "deactivating file completion"
            else
                # Perform file completion
                __%[1]s_debug "activating file completion"
                _arguments '*:filename:_files'
            fi
        elif [ $((directive & %[4]d)) -ne 0 ] && [ ${compCount} -eq 1 ]; then
            __%[1]s_debug "Activating nospace."
            # We can use compadd here as there is no description when
            # there is only one completion.
            compadd -S '' "${lastComp}"
        else
            _describe "completions" completions
        fi
    fi
}

compdef __%[1]s_do_completion %[1]s
`, name, compCmd, BashCompDirectiveError, BashCompDirectiveNoSpace, BashCompDirectiveNoFileComp))
}

// GenZshCompletionV2 generates the zsh completion V2 file and writes to the passed writer.
func (c *Command) GenZshCompletionV2(w io.Writer, includeDesc bool) error {
	buf := new(bytes.Buffer)
	genZshCompV2(buf, c.Name(), includeDesc)
	_, err := buf.WriteTo(w)
	return err
}

// GenZshCompletionFileV2 generates the zsh completion V2 file.
func (c *Command) GenZshCompletionFileV2(filename string, includeDesc bool) error {
	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return c.GenZshCompletionV2(outFile, includeDesc)
}
