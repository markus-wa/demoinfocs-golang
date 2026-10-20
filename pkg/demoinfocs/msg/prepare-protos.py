#!/usr/bin/env python3
"""Prepares the protos in proto/v6 for generate.sh, which runs it first. Idempotent.

- Sets the package com.github.markus_wa.demoinfocs_golang.v6 and prefixes fully qualified type references and
  sibling imports, so the registered names and file paths don't clash with demoinfocs v4's protos
  (Source 1: no package; Source 2: package ...demoinfocs_golang.s2) when a program imports both versions.
- Removes Valve's custom options (extensions of google.protobuf.*Options and their uses): v4 registers the
  same extension numbers, which also clashes, and demoinfocs doesn't read them.
- Removes imports that are unused after that.

String literals and comments are left alone.
"""
import glob
import os
import re

PKG = 'com.github.markus_wa.demoinfocs_golang.v6'

# string literals and comments, which must not be rewritten
_NON_CODE = re.compile(r'"(?:\\.|[^"\\])*"|\'(?:\\.|[^\'\\])*\'|//[^\n]*|/\*.*?\*/', re.S)


def map_code(s, fn):
    """Applies fn to the parts of s that aren't string literals or comments."""
    out, pos = [], 0
    for m in _NON_CODE.finditer(s):
        out.append(fn(s[pos:m.start()]))
        out.append(m.group(0))
        pos = m.end()
    out.append(fn(s[pos:]))
    return ''.join(out)


def split_options(body):
    """Splits the body of a [...] field option list at top-level commas, leaving string literals intact."""
    parts, cur, i = [], '', 0
    while i < len(body):
        m = _NON_CODE.match(body, i)
        if m:
            cur += m.group(0)
            i = m.end()
            continue
        if body[i] == ',':
            parts.append(cur)
            cur = ''
        else:
            cur += body[i]
        i += 1
    parts.append(cur)
    return [p.strip() for p in parts if p.strip()]


def strip_custom_field_options(s):
    # field and enum value options: [default = 32, (key_field) = true] -> [default = 32]
    # The brackets are matched on the original text, so a ']' inside a string literal must not end the list.
    out, i = [], 0
    while True:
        m = re.compile(r'\s*\[').search(s, i)
        if not m:
            out.append(s[i:])
            return ''.join(out)
        # skip matches inside string literals / comments
        inside = next((n for n in _NON_CODE.finditer(s) if n.start() < m.start() < n.end()), None)
        if inside:
            out.append(s[i:inside.end()])
            i = inside.end()
            continue
        j, depth = m.end(), 1
        while depth:
            n = _NON_CODE.match(s, j)
            if n:
                j = n.end()
                continue
            depth += {'[': 1, ']': -1}.get(s[j], 0)
            j += 1
        kept = [p for p in split_options(s[m.end():j - 1]) if not p.startswith('(')]
        out.append(s[i:m.start()])
        if kept:
            out.append(' [' + ', '.join(kept) + ']')
        i = j


_STRING = r'"(?:\\.|[^"\\])*"'


def strip_options(s):
    # whole statements, matched on the full text: their values may be string literals
    s = re.sub(r'^extend \.?google\.protobuf\.\w+Options\s*\{(?:' + _STRING + r'|[^}"])*\}\n?', '', s, flags=re.M)
    s = re.sub(r'^[ \t]*option \([^)]*\)\s*=\s*(?:' + _STRING + r'|[^;"])*;\n', '', s, flags=re.M)
    return s


def prefix_types(code):
    return re.sub(r'(?<=[\s(<,])\.(?!google\.|' + re.escape(PKG) + r'\.)(?=[A-Za-z_])', '.' + PKG + '.', code)


def defined_names(s):
    return set(re.findall(r'^\s*(?:message|enum)\s+(\w+)', map_code(s, lambda c: c), flags=re.M))


def prepare(s):
    s = re.sub(r'^package [^;]*;\n\n?', '', s, flags=re.M)
    s = re.sub(r'^import "(?:s2/|v6/)?(?!google/)([^"/]+\.proto)";', r'import "v6/\1";', s, flags=re.M)
    s = strip_options(s)
    s = strip_custom_field_options(s)
    s = map_code(s, prefix_types)
    # the package goes first (after the syntax line, if any)
    syntax = re.match(r'(syntax\s*=\s*"[^"]*";\n)?', s).group(0)
    return syntax + f'package {PKG};\n\n' + s[len(syntax):].lstrip('\n')


def drop_unused_imports(s, names_by_file):
    code = map_code(s, lambda c: c)

    def used(path):
        if path == 'google/protobuf/descriptor.proto':
            return 'google.protobuf.' in code
        names = names_by_file.get(os.path.basename(path), set())
        return any(re.search(r'\b' + re.escape(n) + r'\b', code) for n in names)

    s = re.sub(r'^import "([^"]+)";\n', lambda m: m.group(0) if used(m.group(1)) else '', s, flags=re.M)

    # removed imports leave blank lines behind: at most one blank line between the header's parts
    body = re.search(r'^(?:message|enum|service|extend)\b', s, flags=re.M)
    head_end = body.start() if body else len(s)
    return re.sub(r'\n{3,}', '\n\n', s[:head_end]) + s[head_end:]


files = sorted(glob.glob('proto/v6/*.proto'))
prepared = {}
for f in files:
    with open(f) as fh:
        prepared[f] = prepare(fh.read())

names_by_file = {os.path.basename(f): defined_names(s) for f, s in prepared.items()}

for f in files:
    new = drop_unused_imports(prepared[f], names_by_file)
    with open(f) as fh:
        old = fh.read()
    if new != old:
        with open(f, 'w') as fh:
            fh.write(new)
