#!/usr/bin/env python3
"""Prepares the protos in proto/v6 for generate.sh. Idempotent; re-run after updating them from upstream.

- Adds the package com.github.markus_wa.demoinfocs_golang.v6 and prefixes fully qualified type references
  and sibling imports, so the registered names and file paths don't clash with demoinfocs v4's protos
  (Source 1: no package; Source 2: package ...demoinfocs_golang.s2) when a program imports both versions.
- Removes Valve's custom options (extensions of google.protobuf.*Options and their uses): v4 registers the
  same extension numbers, which also clashes, and demoinfocs doesn't read them.
"""
import glob
import re

PKG = 'com.github.markus_wa.demoinfocs_golang.v6'


def split_options(body):
    parts, depth, quote, cur = [], 0, False, ''
    for ch in body:
        if ch == '"':
            quote = not quote
        if not quote and ch == ',' and depth == 0:
            parts.append(cur)
            cur = ''
            continue
        cur += ch
    parts.append(cur)
    return [p.strip() for p in parts if p.strip()]


def strip_custom_field_options(s):
    def repl(m):
        kept = [p for p in split_options(m.group(1)) if not p.startswith('(')]
        return ' [' + ', '.join(kept) + ']' if kept else ''
    return re.sub(r'\s*\[([^\[\]]*)\]', repl, s)


def prepare(s):
    s = re.sub(r'^import "(?!google/|s2/|v6/)([^"]+\.proto)";', r'import "v6/\1";', s, flags=re.M)
    s = re.sub(r'(?<=[\s(<])\.(?!google\.|' + re.escape(PKG) + r'\.)(?=[A-Za-z_])', '.' + PKG + '.', s)
    s = re.sub(r'^extend [^{]*\{[^}]*\}\n?', '', s, flags=re.M)
    s = re.sub(r'^\s*option \([^)]*\)[^;]*;\n', '', s, flags=re.M)
    s = strip_custom_field_options(s)
    if not re.search(r'^package ', s, flags=re.M):
        lines = s.split('\n')
        idx = next(i for i, l in enumerate(lines) if l.startswith(('import', 'message', 'enum', 'option', 'service')))
        lines.insert(idx, f'package {PKG};\n')
        s = '\n'.join(lines)
    return s


for f in sorted(glob.glob('proto/v6/*.proto')):
    with open(f) as fh:
        old = fh.read()
    new = prepare(old)
    if new != old:
        with open(f, 'w') as fh:
            fh.write(new)
