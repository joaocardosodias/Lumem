"""Audit Python imports and exact dependency pins without executing notebooks."""

import argparse
import ast
from collections import defaultdict
import json
from pathlib import Path
import subprocess
import sys

from packaging.requirements import Requirement
from packaging.utils import canonicalize_name


ROOT = Path(__file__).resolve().parents[1]
MODULES = {
    'IPython': 'ipython', 'sklearn': 'scikit-learn',
    'psycopg2': 'psycopg2-binary',
}
LOCAL_MODULES = {'app', 'ml_service'}
PROFILES = {
    'modelagem': 'requirements.txt',
    'servico': 'src/modelo/requirements.txt',
    'testes_servico': 'src/modelo/requirements-test.txt',
    'pipeline': 'models/pipeline-sequencial/requirements.txt',
    'automl': 'models/modelo-4/requirements.txt',
}


def read_pins(path):
    pins = {}
    for line in path.read_text(encoding='utf-8-sig').splitlines():
        line = line.strip()
        if not line or line.startswith('#'):
            continue
        if line.startswith('-r '):
            entries = read_pins(path.parent / line[3:].strip())
        else:
            req = Requirement(line)
            specs = list(req.specifier)
            if req.url or len(specs) != 1 or specs[0].operator != '==' or '*' in specs[0].version:
                raise ValueError(f'{path}: dependencia sem versao exata: {line}')
            entries = {canonicalize_name(req.name): specs[0].version}
        for name, version in entries.items():
            if name in pins and pins[name] != version:
                raise ValueError(f'{path}: versoes conflitantes para {name}')
            pins[name] = version
    return pins


def audit():
    paths = subprocess.check_output(
        ['git', 'ls-files', '--cached', '--others', '--exclude-standard', '-z'], cwd=ROOT
    ).decode('utf-8').split('\0')
    paths = sorted({p for p in paths if Path(p).suffix in {'.py', '.ipynb'}})
    inventory = {profile: defaultdict(set) for profile in PROFILES}
    errors = []
    cells_scanned = 0
    for name in paths:
        path = ROOT / name
        text = path.read_text(encoding='utf-8-sig')
        cells = json.loads(text)['cells'] if path.suffix == '.ipynb' else [
            {'cell_type': 'code', 'source': text}
        ]
        profile = 'modelagem'
        if name.startswith('models/modelo-4/'):
            profile = 'automl'
        if name.startswith('src/modelo/'):
            profile = 'testes_servico' if '/tests/' in name else 'servico'
        for index, cell in enumerate(cells):
            if cell['cell_type'] != 'code':
                continue
            cells_scanned += 1
            try:
                tree = ast.parse(''.join(cell['source']))
            except SyntaxError as error:
                errors.append(f'{name}, celula {index}: {error}')
                continue
            location = f'{name}#cell-{index}' if path.suffix == '.ipynb' else name
            dependencies = set()
            for node in ast.walk(tree):
                modules = []
                if isinstance(node, ast.Import):
                    modules = [alias.name for alias in node.names]
                elif isinstance(node, ast.ImportFrom) and node.level == 0 and node.module:
                    modules = [node.module]
                elif isinstance(node, ast.Call):
                    func = node.func
                    method = func.attr if isinstance(func, ast.Attribute) else ''
                    if method in {'read_excel', 'to_excel', 'ExcelFile', 'ExcelWriter'}:
                        dependencies.add('openpyxl')
                    if method in {'read_parquet', 'to_parquet'}:
                        dependencies.add('pyarrow')
                    if method == 'import_module' or isinstance(func, ast.Name) and func.id == '__import__':
                        if node.args and isinstance(node.args[0], ast.Constant) and isinstance(node.args[0].value, str):
                            modules = [node.args[0].value]
                        else:
                            errors.append(f'{location}: import dinamico requer revisao manual')
                for module in modules:
                    module = module.split('.')[0]
                    if module not in sys.stdlib_module_names and module not in LOCAL_MODULES:
                        dependencies.add(canonicalize_name(MODULES.get(module, module)))
            for dependency in dependencies:
                inventory[profile][dependency].add(location)
                if name.startswith('models/pipeline-sequencial/'):
                    inventory['pipeline'][dependency].add(location)
    for profile, filename in PROFILES.items():
        pins = read_pins(ROOT / filename)
        for dependency in inventory[profile]:
            if dependency not in pins:
                errors.append(f'{filename}: falta {dependency}')
    return {
        'files_scanned': len(paths), 'code_cells_and_scripts_scanned': cells_scanned,
        'dependencies': {
            profile: {name: sorted(locations) for name, locations in sorted(deps.items())}
            for profile, deps in inventory.items()
        },
        'errors': errors,
    }


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--json', action='store_true', help='Show import locations per dependency')
    args = parser.parse_args()
    result = audit()
    if args.json:
        print(json.dumps(result, indent=2, ensure_ascii=True))
    else:
        print(f"Arquivos auditados: {result['files_scanned']}")
        print(f"Celulas de codigo e scripts: {result['code_cells_and_scripts_scanned']}")
        for profile, dependencies in result['dependencies'].items():
            print(f"{profile}: {', '.join(dependencies)}")
        for error in result['errors']:
            print(f'ERRO: {error}')
        print('FALHOU' if result['errors'] else 'OK: imports cobertos e versoes exatas.')
    sys.exit(bool(result['errors']))
