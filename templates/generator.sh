#!/bin/sh
set -eu

usage() {
    echo "Usage: $0 [build|configure|dev]"
}

set_colors() {
    FG_BLACK=""
    FG_RED=""
    FG_GREEN=""
    FG_YELLOW=""
    FG_BLUE=""
    FG_MAGENTA=""
    FG_CYAN=""
    FG_WHITE=""
    BOLD=""
    RESET=""

    #shellcheck disable=SC2034
    if tput sgr0 >/dev/null; then
        FG_BLACK=$(tput setaf 0)
        FG_RED=$(tput setaf 1)
        FG_GREEN=$(tput setaf 2)
        FG_YELLOW=$(tput setaf 3)
        FG_BLUE=$(tput setaf 4)
        FG_MAGENTA=$(tput setaf 5)
        FG_CYAN=$(tput setaf 6)
        FG_WHITE=$(tput setaf 7)
        BOLD=$(tput bold)
        RESET=$(tput sgr0)
    fi

    WARNING="$FG_YELLOW"
    ERROR="$FG_RED"
}

# uses curl or wget depending on what is available
download() {
    if [ -z "$1" ]; then
        echo "${ERROR}download() requires a URL as first argument${RESET}"
        exit 1
    fi
    if [ -z "$2" ]; then
        echo "${ERROR}download() requires a destination directory as second argument${RESET}"
        exit 1
    fi
    if [ ! -d "$2" ]; then
        echo "${ERROR}$2 is not a directory${RESET}"
        exit 1
    fi

    if command -v curl >/dev/null; then
        cd "$2" || (echo "${ERROR}Could not cd to $2${RESET}" && exit 1)
        # older versions of curl don't support --output-dir
        curl -sSLO --fail --remote-name "$1"
        cd - >/dev/null
    elif command -v wget >/dev/null; then
        wget -nv -P "$2" "$1"
    else
        echo "${ERROR}Neither curl nor wget is available, cannot download files.${RESET}"
        exit 1
    fi
}

check_dir() {
    if [ ! -d "$1" ]; then
        mkdir "$1"
    fi
}

check_tailwind_bin() {
    if [ ! -f "bin/tailwind" ]; then
        echo "${ERROR}bin/tailwind not found, run $0 configure first${RESET}"
        exit 1
    fi
}

install_tailwind_bin() {
    check_dir "bin"
    if [ ! -f "bin/tailwind" ]; then
        echo "${FG_GREEN}Installing tailwind${RESET}"
        download "https://github.com/tailwindlabs/tailwindcss/releases/download/v3.4.17/tailwindcss-linux-x64" "bin"
        mv bin/tailwindcss-linux-x64 bin/tailwind
        chmod +x bin/tailwind
    fi
}

generate_css() {
    check_dir "dist"
    echo "${FG_GREEN}Generating CSS${RESET}"
    bin/tailwind -i src/input.css -o dist/output.css --minify >/dev/null 2>&1
}

inject_css_into_html() {
    # escape backslashes in CSS
    sed -i 's/\\/\\\\/g' dist/output.css
    # for loop over ban and captcha
    for i in login; do
        # if src/$i.html does not exist, exit
        if [ ! -f "src/$i.html" ]; then
            echo "${ERROR}src/$i.html not found${RESET}"
            exit 1
        fi
        echo "${FG_GREEN}Injecting CSS into dist/$i.html${RESET}"
        # replace <link rel='stylesheet' href='output.css' > with inline style tag in production
        sed "s|<link rel='stylesheet' href='output.css' >|<style>$(sed 's:|:\\|:g' dist/output.css)</style>|" src/$i.html > dist/$i.html
    done
    echo "${FG_GREEN}Cleaning up${RESET}"
    rm dist/output.css
    echo "${FG_GREEN}Done${RESET}"
}

generate_dev_css() {
    echo "${FG_GREEN}Watching for changes to HTML${RESET}"
    bin/tailwind -i src/input.css -o src/output.css --watch >/dev/null 2>&1
}  

# ------------------------------------------------------------------------------

set_colors

if [ $# -lt 1 ]; then
    usage
    exit 1
fi

action="$1"
shift

case $action in
    build | configure | dev)
        ;;
    *)
        echo "${ERROR}Unknown action: $action${RESET}"
        usage
        exit 1
        ;;
esac

case $action in
    build)
        check_tailwind_bin
        generate_css
        inject_css_into_html
        ;;
    configure)
        install_tailwind_bin
        ;;
    dev)
        check_tailwind_bin
        generate_dev_css
        ;;
esac
