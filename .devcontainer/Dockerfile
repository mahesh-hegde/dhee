FROM golang:1.26-trixie

ARG USERNAME=dev
ARG USER_UID=1000
ARG USER_GID=$USER_UID

ENV DEV_USERNAME=$USERNAME
ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get upgrade -y && apt-get install -y shellcheck \
    ipython3 irssi python3-pip lcov httpie cloc tree pipx bash-completion \
    fzf fd-find ripgrep apt-file jq vim man less sudo zsh gh && apt-get clean

# https://code.visualstudio.com/remote/advancedcontainers/add-nonroot-user#_creating-a-non-root-user
RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

USER ${USERNAME}
WORKDIR /home/${USERNAME}

RUN pipx install uv

# Setup bash history: https://code.visualstudio.com/remote/advancedcontainers/persist-bash-history
RUN SNIPPET="export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
    && mkdir /home/${USERNAME}/commandhistory \
    && touch /home/${USERNAME}/commandhistory/.bash_history \
    && echo "$SNIPPET" >> "/home/${USERNAME}/.bashrc"

COPY devcontainer.bashrc /home/${USERNAME}/.bashrc
