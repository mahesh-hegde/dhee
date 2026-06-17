#!/bin/bash

# If not running interactively, don't do anything
case $- in
    *i*) ;;
      *) return;;
esac

HISTCONTROL=ignoreboth
shopt -s histappend
HISTSIZE=100000
HISTTIMEFORMAT="%c: "

# check the window size after each command and, if necessary,
# update the values of LINES and COLUMNS.
shopt -s checkwinsize
# If set, the pattern "**" used in a pathname expansion context will
# match all files and zero or more directories and subdirectories.
shopt -s globstar
# make less more friendly for non-text input files, see lesspipe(1)
[ -x /usr/bin/lesspipe ] && eval "$(SHELL=/bin/sh lesspipe)"

if [[ "$OSTYPE" = "linux-gnu" ]]; then
    export PS1='\[\033[01;32m\]\u\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\] \n$ '
fi

# enable color support of ls and also add handy aliases
if [ -x /usr/bin/dircolors ]; then
    test -r ~/.dircolors && eval "$(dircolors -b ~/.dircolors)" || eval "$(dircolors -b)"
    alias ls='ls --color=auto'
    alias dir='dir --color=auto'
    #alias vdir='vdir --color=auto'
    alias grep='grep --color=auto'
    alias fgrep='fgrep --color=auto'
    alias egrep='egrep --color=auto'
fi

## Make prompt display error code
prompt_show_ec () {
 # Catch exit code
 ec=$?
 # Display exit code in red text unless zero
 if [ $ec -ne 0 ];then
  echo -e "\033[31m[$ec]\033[0m"
 fi
}
PROMPT_COMMAND="prompt_show_ec; $PROMPT_COMMAND"

export PAGER=less

# git aliases
alias Gs='git status -s'
alias Gl='git log'
alias Gls='git log --stat'
alias Gd='git diff --minimal'
alias Gds='git diff --staged --minimal'
alias Gsw='git show'
alias Gp='git push'

# Options to fzf command
export FZF_COMPLETION_OPTS='--border --info=inline'

# Use fd (https://github.com/sharkdp/fd) instead of the default find
# command for listing path candidates.
# - The first argument to the function ($1) is the base path to start traversal
# - See the source code (completion.{bash,zsh}) for the details.
_fzf_compgen_path() {
  fdfind --hidden --follow --exclude ".git" . "$1"
}

# Use fd to generate the list for directory completion
_fzf_compgen_dir() {
  fdfind --type d --hidden --follow --exclude ".git" . "$1"
}

if [[ "$OSTYPE" = "linux-gnu" ]];
then
	_fzf_completion_script=/usr/share/doc/fzf/examples/completion.bash
	[[ -f $_fzf_completion_script ]] && source $_fzf_completion_script
	source /usr/share/doc/fzf/examples/key-bindings.bash
fi

if which gh 2> /dev/null; then
  eval "$(gh completion -s bash)"
fi

export USE_PROMPTSYNTH=${USE_PROMPTSYNTH:-1}
export USE_BASH_GIT_PROMPT=${USE_BASH_GIT_PROMPT:-0}
if [[ -f "$HOME/.local/bin/promptsynth" ]] && [[ "$USE_PROMPTSYNTH" -ne 0 ]]
then
	__remove_suffix=' \n$ '
	PS1_NONL="${PS1%"${__remove_suffix}"}"
	export PS1="${PS1_NONL} "'$(promptsynth)\n$ '
elif [[ -f "$HOME/.local/bash-git-prompt/gitprompt.sh" ]] && [[ "$USE_BASH_GIT_PROMPT" -ne 0 ]]
then
	GIT_PROMPT_FETCH_REMOTE_STATUS=0
	GIT_PROMPT_ONLY_IN_REPO=1
	source ~/.local/bash-git-prompt/gitprompt.sh
	## use clone command: git clone https://github.com/magicmonty/bash-git-prompt.git ~/.bash-git-prompt --depth=1
else
	echo "Using default git prompt"
	source /etc/bash_completion.d/git-prompt
	export GIT_PS1_SHOWDIRTYSTATE=1
	export GIT_PS1_SHOWUPSTREAM="verbose"
	export GIT_PS1_SHOWCOLORHINTS=1
	export GIT_PS1_SHOWUNTRACKEDFILES=1
	__remove_suffix=' \n$ '
	PS1_NONL="${PS1%"${__remove_suffix}"}"
	export PS1="${PS1_NONL} "'${orange}$(__git_ps1 [%s])${default} \n$ '
fi
export GIT_PROMPT_THEME="Custom"

if [[ -e /usr/bin/fdfind ]]; then
	alias fd="fdfind"
fi

export EDITOR="code --wait --new-window"

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"  # This loads nvm
[ -s "$NVM_DIR/bash_completion" ] && \. "$NVM_DIR/bash_completion"  # This loads nvm bash_completion
