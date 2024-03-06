#!/usr/bin/env python3
import os
import re
import argparse
import subprocess

# List of modules to update
MODULES = [
    'optimism',
    'simulators/optimism/l1ops',
    'simulators/optimism/p2p',
    'simulators/optimism/rpc',
    'simulators/optimism/daisy-chain',
]

# Regex pattern to match the line to be replaced in go.mod
REPLACER_RE = r'replace github\.com/ethereum/go-ethereum (.*) => github.com/ethereum-optimism/op-geth'

# Regex pattern to extract the version from a dependency line
VERSION_RE = r'github\.com/ethereum-optimism/op-geth@([va-f0-9\d\.\-]+)'

# Argument parser setup
parser = argparse.ArgumentParser()
parser.add_argument('--version', help='version to upgrade to', required=True)
parser.add_argument('--geth', action=argparse.BooleanOptionalAction, help='update geth rather than op dependencies')

def main():
    args = parser.parse_args()
    if args.geth:
        update_geth(args)
    else:
        update_op_deps(args)

def update_geth(args):
    for mod in MODULES:
        # Check if the module needs to be updated
        should_update = check_module_for_update(mod, REPLACER_RE)
        if not should_update:
            continue
        # Update the module
        print(f'Updating {mod}')
        run_update_command(mod, args.version)
        tidy(mod)

def update_op_deps(args):
    for mod in MODULES:
        # Identify dependencies that need updating
        needs = identify_dependencies(mod)
        if not needs:
            continue
        print(f'Updating {mod}')
        for need in needs:
            go_get(mod, need, args.version)
        tidy(mod)

def check_module_for_update(mod, pattern):
    """Check if a module needs to be updated based on a regex pattern."""
    with open(os.path.join(mod, 'go.mod')) as f:
        for line in f:
            if re.search(pattern, line):
                return True
    return False

def run_update_command(mod, version):
    """Run the command to update a module."""
    # Example command to update the module
    # Adjust based on the actual update process
    command = [
        'go', 'mod', 'edit', '-replace',
        f'github.com/ethereum/go-ethereum@{version}=github.com/ethereum-optimism/op-geth@{version}'
    ]
    run(command, cwd=mod, check=True)

def identify_dependencies(mod):
    """Identify dependencies in a module that need to be updated."""
    needs = set()
    with open(os.path.join(mod, 'go.mod')) as f:
        for line in f:
            if line.endswith('// indirect\n'):
                continue
            if not re.search(r'github.com/ethereum-optimism/optimism', line):
                continue
            dep = line.strip().split(' ')[0]
            needs.add(dep)
    return needs

def go_get(mod, dep, version, capture_output=False, check=True):
    """Run the go get command to update a dependency."""
    command = ['go', 'get', f'{dep}@{version}']
    run(command, cwd=mod, check=check, capture_output=capture_output)

def tidy(mod):
    """Run the go mod tidy command to clean up the module."""
    command = ['go', 'mod', 'tidy']
    run(command, cwd=mod, check=True)

def run(args, cwd=None, capture_output=False, check=True):
    """Run a subprocess command with the given arguments."""
    print(subprocess.list2cmdline(args))
    return subprocess.run(args, cwd=cwd, check=check, capture_output=capture_output)

if __name__ == '__main__':
    main()
