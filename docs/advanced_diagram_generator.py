#!/usr/bin/env python3
"""
Generador Avanzado de Diagramas Mermaid
========================================

Script avanzado para extraer y generar imágenes PNG de diagramas Mermaid
desde archivos de documentación usando kroki.io API.

Características:
- Detección inteligente de tipos de diagramas
- Generación de nombres descriptivos automáticos
- Soporte para múltiples formatos de salida
- Generación de índice HTML con todas las imágenes
- Validación de sintaxis Mermaid
- Reporte detallado de procesamiento
- Manejo de errores robusto
"""

import os
import re
import requests
import base64
import zlib
import json
import hashlib
import time
from pathlib import Path
from typing import List, Dict, Tuple, Optional
from dataclasses import dataclass, field
from datetime import datetime
import argparse

@dataclass
class DiagramInfo:
    """Información de un diagrama extraído."""
    code: str
    title: str
    type: str
    line_number: int
    context: str
    file_path: str
    hash: str
    output_filename: Optional[str] = field(default=None)
    
class AdvancedDiagramGenerator:
    """Generador avanzado de diagramas con funcionalidades extendidas."""
    
    def __init__(self, base_url: str = "https://kroki.io", output_format: str = "png", skip_existing: bool = True, force: bool = False, clean: bool = False, clean_only: bool = False):
        self.base_url = base_url
        self.output_format = output_format
        self.skip_existing = skip_existing
        self.force = force
        self.clean = clean
        self.clean_only = clean_only
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'AdvancedDiagramGenerator/2.0',
            'Accept': f'image/{output_format}'
        })
        
        # Estadísticas de procesamiento
        self.stats = {
            'files_processed': 0,
            'diagrams_found': 0,
            'diagrams_generated': 0,
            'errors': 0,
            'skipped': 0
        }
        
        # Registro de diagramas procesados
        self.processed_diagrams: List[DiagramInfo] = []

        # Razones de error agregadas para diagnóstico
        self.error_reasons = {
            'parentheses_in_labels': 0,
            'syntax_error': 0,
            'connection_error': 0,
            'other': 0
        }
        
        # Patrones para detectar tipos de diagramas
        self.diagram_patterns = {
            'flowchart': [
                r'flowchart\s+(TD|TB|BT|RL|LR)',
                r'graph\s+(TD|TB|BT|RL|LR)',
                r'flowchart',
                r'graph'
            ],
            'sequence': [
                r'sequenceDiagram',
                r'participant\s+\w+',
                r'\w+\s*->>?\s*\w+'
            ],
            'class': [
                r'classDiagram',
                r'class\s+\w+',
                r'\w+\s*:\s*\w+'
            ],
            'state': [
                r'stateDiagram-v2',
                r'stateDiagram',
                r'state\s+\w+'
            ],
            'er': [
                r'erDiagram',
                r'\w+\s*\|\|--\|\|\s*\w+',
                r'\w+\s*\}o--o\{\s*\w+'
            ],
            'journey': [
                r'journey',
                r'title\s+.+',
                r'section\s+.+'
            ],
            'gantt': [
                r'gantt',
                r'dateFormat\s+',
                r'section\s+.+'
            ],
            'pie': [
                r'pie\s+title',
                r'pie',
                r'"\w+"\s*:\s*\d+'
            ],
            'gitgraph': [
                r'gitGraph',
                r'commit',
                r'branch\s+\w+'
            ]
        }
    
    def detect_diagram_type(self, code: str) -> str:
        """Detecta el tipo de diagrama basándose en patrones."""
        code_clean = re.sub(r'%%.*', '', code)  # Remover comentarios
        
        for diagram_type, patterns in self.diagram_patterns.items():
            for pattern in patterns:
                if re.search(pattern, code_clean, re.IGNORECASE | re.MULTILINE):
                    return diagram_type
        
        return 'unknown'
    
    def extract_title_from_code(self, code: str, diagram_type: str) -> Optional[str]:
        """Extrae título del código del diagrama."""
        # Patrones para extraer títulos
        title_patterns = [
            r'title\s*[:\s]+(.+)',
            r'%%\s*title[:\s]+(.+)',
            r'%%\s*(.+)',
            r'#\s*(.+)',
        ]
        
        for pattern in title_patterns:
            match = re.search(pattern, code, re.IGNORECASE)
            if match:
                title = match.group(1).strip()
                if len(title) > 2:
                    return title
        
        # Extraer información específica por tipo
        if diagram_type == 'flowchart':
            return self._extract_flowchart_title(code)
        elif diagram_type == 'sequence':
            return self._extract_sequence_title(code)
        elif diagram_type == 'journey':
            return self._extract_journey_title(code)
        
        return None
    
    def _extract_flowchart_title(self, code: str) -> Optional[str]:
        """Extrae título de un flowchart."""
        # Buscar el primer nodo con texto descriptivo
        node_patterns = [
            r'\[([^\]]{4,})\]',  # Nodos rectangulares
            r'\(([^\)]{4,})\)',  # Nodos circulares
            r'\{([^\}]{4,})\}',  # Nodos rombo
        ]
        
        for pattern in node_patterns:
            match = re.search(pattern, code)
            if match:
                text = match.group(1).strip()
                if not text.isdigit() and len(text) > 3:
                    return text
        
        return None
    
    def _extract_sequence_title(self, code: str) -> Optional[str]:
        """Extrae título de un diagrama de secuencia."""
        # Buscar participantes
        participants = re.findall(r'participant\s+(\w+)', code, re.IGNORECASE)
        if len(participants) >= 2:
            return f"Secuencia {participants[0]} - {participants[1]}"
        
        return None
    
    def _extract_journey_title(self, code: str) -> Optional[str]:
        """Extrae título de un user journey."""
        match = re.search(r'title\s+(.+)', code, re.IGNORECASE)
        if match:
            return match.group(1).strip()
        
        return None
    
    def generate_filename(self, diagram: DiagramInfo, index: int) -> str:
        """Genera nombre de archivo para el diagrama."""
        # Obtener nombre del archivo fuente sin extensión
        source_file_name = os.path.splitext(os.path.basename(diagram.file_path))[0]
        
        # Usar título si está disponible
        if diagram.title and diagram.title != 'unknown':
            diagram_name = self._clean_filename(diagram.title)
        else:
            # Generar basado en contexto y tipo
            context_clean = self._clean_filename(diagram.context) if diagram.context else 'diagram'
            diagram_name = f"{context_clean}_{diagram.type}_{index}"
        
        # Agregar hash para evitar colisiones
        hash_suffix = diagram.hash[:8]
        
        # Formato: ARCHIVO_FUENTE_nombre_diagrama_hash.formato
        return f"{source_file_name}_{diagram_name}_{hash_suffix}.{self.output_format}"
    
    def _clean_filename(self, text: str) -> str:
        """Limpia texto para usar como nombre de archivo."""
        # Remover caracteres especiales
        text = re.sub(r'[^\w\s-]', '', text)
        # Reemplazar espacios con guiones bajos
        text = re.sub(r'\s+', '_', text.strip())
        # Convertir a minúsculas
        text = text.lower()
        # Limitar longitud
        if len(text) > 40:
            text = text[:40]
        
        return text
    
    def extract_diagrams_from_file(self, file_path: str) -> List[DiagramInfo]:
        """Extrae todos los diagramas de un archivo."""
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
        except Exception as e:
            print(f"❌ Error leyendo {file_path}: {e}")
            return []
        
        diagrams = []
        lines = content.split('\n')
        current_section = ""
        
        i = 0
        while i < len(lines):
            line = lines[i].strip()
            
            # Detectar secciones para contexto
            if line.startswith('#'):
                current_section = line.strip('#').strip()
                i += 1
                continue
            
            # Buscar bloques Mermaid
            if line.startswith('```mermaid'):
                diagram_start = i + 1
                diagram_lines = []
                i += 1
                
                # Extraer contenido del bloque
                while i < len(lines) and not lines[i].strip().startswith('```'):
                    diagram_lines.append(lines[i])
                    i += 1
                
                if diagram_lines:
                    mermaid_code = '\n'.join(diagram_lines).strip()
                    
                    if mermaid_code:  # Solo procesar si hay contenido
                        diagram_type = self.detect_diagram_type(mermaid_code)
                        title = self.extract_title_from_code(mermaid_code, diagram_type)
                        
                        if not title:
                            title = f"{current_section}_{diagram_type}" if current_section else f"{diagram_type}_diagram"
                        
                        # Generar hash único
                        code_hash = hashlib.md5(mermaid_code.encode()).hexdigest()
                        
                        diagram = DiagramInfo(
                            code=mermaid_code,
                            title=title,
                            type=diagram_type,
                            line_number=diagram_start,
                            context=current_section,
                            file_path=file_path,
                            hash=code_hash
                        )
                        
                        diagrams.append(diagram)
            
            i += 1
        
        return diagrams
    
    def validate_mermaid_syntax(self, code: str) -> Tuple[bool, str]:
        """Valida sintaxis básica de Mermaid (tolerante a directivas/comentarios)."""
        try:
            if not code or not code.strip():
                return False, "Código vacío"

            # Saltar líneas vacías y directivas/comentarios de Mermaid (%% ...)
            first_line = None
            for raw in code.split('\n'):
                line = raw.strip()
                if not line:
                    continue
                if line.startswith('%%'):
                    # directiva o comentario, saltar
                    continue
                first_line = line
                break

            if not first_line:
                return False, "No se encontró contenido válido en el bloque"

            # Tipos válidos conocidos (ampliado)
            valid_starts = [
                'flowchart', 'graph', 'sequenceDiagram', 'classDiagram',
                'stateDiagram', 'stateDiagram-v2', 'erDiagram', 'journey',
                'gantt', 'pie', 'gitGraph', 'mindmap', 'timeline',
                'requirementDiagram'
            ]

            if not any(first_line.lower().startswith(start.lower()) for start in valid_starts):
                return False, f"Tipo de diagrama no reconocido: {first_line}"

            return True, "Sintaxis válida"
        except Exception as e:
            return False, f"Error de validación: {e}"
    
    def _preprocess_mermaid(self, code: str) -> str:
        """Normaliza casos que rompen Mermaid: etiquetas con saltos de línea dentro de [], {}, ().
        Reemplaza los \n internos por <br/> sólo dentro del contenido de la etiqueta.
        """
        processed = code
        patterns = [
            (r"\[([^\]]*?)\]", "[", "]"),
            (r"\{([^\}]*?)\}", "{", "}"),
            (r"\(([^\)]*?)\)", "(", ")"),
        ]
        for regex, open_ch, close_ch in patterns:
            def repl(m: re.Match) -> str:
                inner = m.group(1)
                if "\n" in inner:
                    replaced = inner.replace('\n', '<br/>')
                    return f"{open_ch}{replaced}{close_ch}"
                return m.group(0)
            processed = re.sub(regex, repl, processed, flags=re.DOTALL)
        return processed

    def _sanitize_label_parentheses(self, code: str) -> str:
        """Escapa paréntesis dentro de etiquetas de nodos/decisiones ([], {})."""
        processed = code
        # Para []
        def repl_square(m: re.Match) -> str:
            inner = m.group(1)
            inner = inner.replace('(', '&#40;').replace(')', '&#41;')
            return f"[{inner}]"
        processed = re.sub(r"\[([^\]]*?)\]", repl_square, processed, flags=re.DOTALL)
        # Para {}
        def repl_brace(m: re.Match) -> str:
            inner = m.group(1)
            inner = inner.replace('(', '&#40;').replace(')', '&#41;')
            return f"{{{inner}}}"
        processed = re.sub(r"\{([^\}]*?)\}", repl_brace, processed, flags=re.DOTALL)
        return processed

    def generate_image_kroki(self, diagram: DiagramInfo, output_path: str) -> bool:
        """Genera imagen usando kroki.io API utilizando POST text/plain (con saneamiento)."""
        try:
            # Validar sintaxis primero (tolerante)
            is_valid, validation_msg = self.validate_mermaid_syntax(diagram.code)
            if not is_valid:
                print(f"  ⚠️  Sintaxis inválida: {validation_msg}")
                self.error_reasons['syntax_error'] += 1
                return False

            url = f"{self.base_url}/mermaid/{self.output_format}"
            print(f"  🔄 Generando: {os.path.basename(output_path)}")

            headers = {
                'Content-Type': 'text/plain; charset=utf-8',
                'Accept': f'image/{self.output_format}',
                'User-Agent': 'AdvancedDiagramGenerator/2.3'
            }

            # Preprocesar código para reemplazar saltos de línea en etiquetas
            processed_code = self._preprocess_mermaid(diagram.code)

            response = self.session.post(url, data=processed_code.encode('utf-8'), headers=headers, timeout=60)

            if response.status_code == 200:
                os.makedirs(os.path.dirname(output_path), exist_ok=True)
                with open(output_path, 'wb') as f:
                    f.write(response.content)
                print(f"  ✅ Generado: {output_path}")
                return True

            # Intentar obtener mensaje JSON legible
            jheaders = headers.copy()
            jheaders['Accept'] = 'application/json'
            jresp = None
            err_message = None
            try:
                jresp = self.session.post(url, data=processed_code.encode('utf-8'), headers=jheaders, timeout=60)
                if jresp is not None and jresp.headers.get('Content-Type','').startswith('application/json'):
                    err = jresp.json()
                    err_message = err.get('error', {}).get('message') or str(err)
            except Exception:
                pass

            # Si parece ser por paréntesis en etiquetas, intentar saneamiento y reintentar
            retried = False
            if err_message and "got 'PS'" in err_message:
                self.error_reasons['parentheses_in_labels'] += 1
                sanitized = self._sanitize_label_parentheses(processed_code)
                try:
                    response2 = self.session.post(url, data=sanitized.encode('utf-8'), headers=headers, timeout=60)
                    retried = True
                    if response2.status_code == 200:
                        os.makedirs(os.path.dirname(output_path), exist_ok=True)
                        with open(output_path, 'wb') as f:
                            f.write(response2.content)
                        print(f"  ✅ Generado (saneado paréntesis en etiquetas): {output_path}")
                        return True
                except Exception:
                    pass

            # Guardar imagen de error si vino como PNG
            content_type = response.headers.get('Content-Type', '')
            if 'image/png' in content_type or (hasattr(response, 'content') and response.content[:4] == b'\x89PNG'):
                err_path = self._error_image_path(output_path)
                try:
                    os.makedirs(os.path.dirname(err_path), exist_ok=True)
                    with open(err_path, 'wb') as f:
                        f.write(response.content)
                    print(f"  ❌ Error HTTP {response.status_code}. Imagen de error guardada: {err_path}")
                except Exception as se:
                    print(f"  ❌ Error guardando imagen de error: {se}")

            # Mensaje de detalle
            if err_message:
                print(f"  ❌ Detalle: {err_message}")
                if "got 'PS'" in err_message and not retried:
                    print("  💡 Posible causa: paréntesis sin escapar dentro de etiquetas ([], {}). Se recomienda usar &#40; y &#41; o escribir el texto sin paréntesis dentro de la etiqueta.")
            else:
                preview = response.text[:200] if hasattr(response, 'text') else ''
                print(f"  ❌ Error HTTP {response.status_code}. Respuesta: {preview}")

            # Clasificar error
            if err_message:
                if "got 'PS'" in err_message:
                    self.error_reasons['parentheses_in_labels'] += 1
                else:
                    self.error_reasons['syntax_error'] += 1
            else:
                self.error_reasons['other'] += 1

            return False
        except requests.exceptions.RequestException as e:
            print(f"  ❌ Error de conexión: {e}")
            self.error_reasons['connection_error'] += 1
            return False
        except Exception as e:
            print(f"  ❌ Error inesperado: {e}")
            self.error_reasons['other'] += 1
            return False

    def _error_image_path(self, output_path: str) -> str:
        root, ext = os.path.splitext(output_path)
        return f"{root}_error{ext}"
    
    def process_file(self, file_path: str) -> int:
        """Procesa un archivo y genera imágenes de sus diagramas."""
        print(f"\n📄 Procesando: {file_path}")
        
        diagrams = self.extract_diagrams_from_file(file_path)
        
        if not diagrams:
            print("   ℹ️  Sin diagramas encontrados")
            return 0
        
        print(f"   📊 Encontrados {len(diagrams)} diagramas")
        
        success_count = 0
        file_dir = os.path.dirname(file_path)
        source_file_name = os.path.splitext(os.path.basename(file_path))[0]
        
        for i, diagram in enumerate(diagrams, 1):
            # Generar nombre de archivo
            filename = self.generate_filename(diagram, i)
            diagram.output_filename = filename
            output_path = os.path.join(file_dir, filename)
            
            print(f"   📊 Diagrama {i}: {diagram.title}")
            print(f"      Tipo: {diagram.type}")
            print(f"      Guardando como: {filename}")
            print(f"      Ruta: {output_path}")

            # Omitir si ya existe (a menos que force)
            if self.skip_existing and not self.force and os.path.exists(output_path):
                print(f"      ♻️  Ya existe, se omite: {filename}")
                self.processed_diagrams.append(diagram)
                # No sumamos 'diagrams_generated', pero sí 'diagrams_found'
                continue
            
            # Generar imagen
            if self.generate_image_kroki(diagram, output_path):
                success_count += 1
                self.processed_diagrams.append(diagram)
                self.stats['diagrams_generated'] += 1
            else:
                self.stats['errors'] += 1
            
            # Pausa para no sobrecargar la API
            time.sleep(0.5)
        
        self.stats['diagrams_found'] += len(diagrams)
        return success_count
    
    def find_documentation_files(self, docs_dir: str, pattern: str = None) -> List[str]:
        """Encuentra archivos de documentación que puedan contener diagramas."""
        md_files = []
        docs_path = Path(docs_dir)
        
        if not docs_path.exists():
            print(f"❌ Directorio {docs_dir} no existe")
            return []
        
        # Buscar archivos .md
        for md_file in docs_path.rglob("*.md"):
            file_path = str(md_file)
            
            # Aplicar filtro de patrón si se especifica
            if pattern and pattern.lower() not in file_path.lower():
                continue
            
            # Filtrar archivos que probablemente contengan diagramas
            file_name = md_file.name.lower()
            file_content_keywords = [
                'diagrama', 'flujo', 'flow', 'sequence', 'chart',
                'arquitectura', 'overview', 'eventos', 'registro',
                'mermaid', 'graph'
            ]
            
            # Incluir si el nombre contiene palabras clave
            if any(keyword in file_name for keyword in file_content_keywords):
                md_files.append(file_path)
                continue
            
            # O si la ruta contiene palabras clave
            if any(keyword in file_path.lower() for keyword in file_content_keywords):
                md_files.append(file_path)
                continue
            
            # O verificar contenido del archivo para bloques mermaid
            try:
                with open(md_file, 'r', encoding='utf-8') as f:
                    content = f.read()
                    if '```mermaid' in content:
                        md_files.append(file_path)
            except:
                pass  # Ignorar errores de lectura
        
        return sorted(md_files)
    
    def generate_html_index(self, output_dir: str = "docs") -> str:
        """Genera un índice HTML con todas las imágenes generadas."""
        if not self.processed_diagrams:
            return ""
        
        html_content = f"""
<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Índice de Diagramas - Campaing App Chat</title>
    <style>
        body {{
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            line-height: 1.6;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }}
        .container {{
            max-width: 1200px;
            margin: 0 auto;
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }}
        h1 {{
            color: #2c3e50;
            text-align: center;
            border-bottom: 3px solid #3498db;
            padding-bottom: 10px;
        }}
        .stats {{
            background: #ecf0f1;
            padding: 15px;
            border-radius: 5px;
            margin: 20px 0;
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 10px;
        }}
        .stat-item {{
            text-align: center;
        }}
        .stat-number {{
            font-size: 2em;
            font-weight: bold;
            color: #3498db;
        }}
        .diagram-grid {{
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
            gap: 20px;
            margin-top: 30px;
        }}
        .diagram-card {{
            border: 1px solid #ddd;
            border-radius: 8px;
            padding: 15px;
            background: #fafafa;
        }}
        .diagram-title {{
            font-weight: bold;
            color: #2c3e50;
            margin-bottom: 10px;
        }}
        .diagram-info {{
            font-size: 0.9em;
            color: #7f8c8d;
            margin-bottom: 15px;
        }}
        .diagram-image {{
            max-width: 100%;
            height: auto;
            border: 1px solid #ddd;
            border-radius: 4px;
        }}
        .file-path {{
            font-family: monospace;
            background: #e8e8e8;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 0.8em;
        }}
        .diagram-type {{
            display: inline-block;
            background: #3498db;
            color: white;
            padding: 2px 8px;
            border-radius: 12px;
            font-size: 0.8em;
            margin-right: 10px;
        }}
    </style>
</head>
<body>
    <div class="container">
        <h1>📊 Índice de Diagramas - Campaing App Chat</h1>
        
        <div class="stats">
            <div class="stat-item">
                <div class="stat-number">{self.stats['files_processed']}</div>
                <div>Archivos Procesados</div>
            </div>
            <div class="stat-item">
                <div class="stat-number">{self.stats['diagrams_found']}</div>
                <div>Diagramas Encontrados</div>
            </div>
            <div class="stat-item">
                <div class="stat-number">{self.stats['diagrams_generated']}</div>
                <div>Imágenes Generadas</div>
            </div>
            <div class="stat-item">
                <div class="stat-number">{self.stats['errors']}</div>
                <div>Errores</div>
            </div>
        </div>
        
        <p><strong>Generado:</strong> {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
        
        <div class="diagram-grid">
"""
        
        # Agrupar diagramas por archivo
        diagrams_by_file = {}
        for diagram in self.processed_diagrams:
            file_path = diagram.file_path
            if file_path not in diagrams_by_file:
                diagrams_by_file[file_path] = []
            diagrams_by_file[file_path].append(diagram)
        
        # Generar cards para cada diagrama
        for file_path, diagrams in diagrams_by_file.items():
            for diagram in diagrams:
                # Generar ruta relativa de la imagen usando el nombre realmente generado
                file_dir = os.path.dirname(diagram.file_path)
                filename = diagram.output_filename or self.generate_filename(diagram, 1)
                image_path = os.path.relpath(os.path.join(file_dir, filename), output_dir)
                
                html_content += f"""
            <div class="diagram-card">
                <div class="diagram-title">{diagram.title}</div>
                <div class="diagram-info">
                    <span class="diagram-type">{diagram.type}</span>
                    <span class="file-path">{os.path.relpath(diagram.file_path, output_dir)}</span>
                </div>
                <img src="{image_path}" alt="{diagram.title}" class="diagram-image" 
                     onerror="this.style.display='none'; this.nextElementSibling.style.display='block';">
                <div style="display:none; color: #e74c3c; text-align: center; padding: 20px;">
                    ❌ Error cargando imagen
                </div>
            </div>
"""
        
        html_content += """
        </div>
    </div>
</body>
</html>
"""
        
        # Guardar archivo HTML
        index_path = os.path.join(output_dir, "diagrams_index.html")
        with open(index_path, 'w', encoding='utf-8') as f:
            f.write(html_content)
        
        return index_path
    
    def _is_generated_image(self, file_name: str) -> bool:
        return re.match(r"^.+_[0-9a-fA-F]{8}\.(png|svg|pdf)$", file_name) is not None

    def _clean_generated(self, docs_dir: str, pattern: Optional[str] = None) -> int:
        print("🧹 Iniciando limpieza de imágenes generadas...")
        removed = 0
        # Determinar directorios candidatos a limpiar
        md_files = self.find_documentation_files(docs_dir, pattern)
        dirs = sorted(set(os.path.dirname(p) for p in md_files)) if md_files else [docs_dir]

        for d in dirs:
            try:
                for entry in os.scandir(d):
                    if not entry.is_file():
                        continue
                    name = entry.name
                    if self._is_generated_image(name):
                        try:
                            os.remove(entry.path)
                            removed += 1
                            print(f"   🗑️  Eliminado: {entry.path}")
                        except Exception as de:
                            print(f"   ⚠️  No se pudo eliminar {entry.path}: {de}")
            except FileNotFoundError:
                continue
        # Eliminar índice HTML
        index_path = os.path.join(docs_dir, "diagrams_index.html")
        if os.path.exists(index_path):
            try:
                os.remove(index_path)
                removed += 1
                print(f"   🗑️  Eliminado: {index_path}")
            except Exception as de:
                print(f"   ⚠️  No se pudo eliminar {index_path}: {de}")
        print(f"🧹 Limpieza completada. Archivos eliminados: {removed}")
        return removed

    def generate_all_diagrams(self, docs_dir: str = "docs", pattern: str = None) -> Dict[str, int]:
        """Genera todas las imágenes de diagramas."""
        print("🚀 GENERADOR AVANZADO DE DIAGRAMAS MERMAID")
        print("=" * 50)
        print(f"📁 Directorio: {docs_dir}")
        print(f"🎯 Formato: {self.output_format}")
        if pattern:
            print(f"🔍 Filtro: {pattern}")
        print("=" * 50)

        # Limpieza previa si está activada
        if self.clean or self.clean_only:
            self._clean_generated(docs_dir, pattern)
            if self.clean_only:
                print("🧹 Modo clean-only: se realizó la limpieza y se finaliza sin generar.")
                return self.stats
        
        # Verificar conectividad
        try:
            response = requests.get(self.base_url, timeout=10)
            print(f"✅ Conectado a {self.base_url}")
        except:
            print(f"❌ Error conectando a {self.base_url}")
            return self.stats
        
        # Encontrar archivos
        md_files = self.find_documentation_files(docs_dir, pattern)
        
        if not md_files:
            print("❌ No se encontraron archivos de documentación")
            return self.stats
        
        print(f"📋 Encontrados {len(md_files)} archivos para procesar")
        
        # Procesar archivos
        for file_path in md_files:
            try:
                self.process_file(file_path)
                self.stats['files_processed'] += 1
            except Exception as e:
                print(f"❌ Error procesando {file_path}: {e}")
                self.stats['errors'] += 1
        
        # Generar índice HTML
        if self.processed_diagrams:
            index_path = self.generate_html_index(docs_dir)
            print(f"\n📄 Índice HTML generado: {index_path}")
        
        # Mostrar resumen final
        print(f"\n📊 RESUMEN FINAL")
        print("=" * 30)
        print(f"📄 Archivos procesados: {self.stats['files_processed']}")
        print(f"📊 Diagramas encontrados: {self.stats['diagrams_found']}")
        print(f"🖼️  Imágenes generadas: {self.stats['diagrams_generated']}")
        print(f"❌ Errores: {self.stats['errors']}")
        print("=" * 30)
        
        if self.stats['diagrams_generated'] > 0:
            print("✅ ¡Generación completada exitosamente!")

        # Diagnóstico agregado de errores
        if self.stats['errors'] > 0:
            print("\n🧪 DIAGNÓSTICO DE ERRORES FRECUENTES")
            print("-" * 34)
            if self.error_reasons['parentheses_in_labels'] > 0:
                print(f"🔸 Paréntesis en etiquetas ([], {{}}) sin escapar: {self.error_reasons['parentheses_in_labels']}")
                print("   Sugerencia: dentro de etiquetas, reemplace '(' por '&#40;' y ')' por '&#41;'.")
                print("   Ej.: A[Parsear flags &#40;-u -t -r&#41;]")
            if self.error_reasons['syntax_error'] > 0:
                print(f"🔸 Otros errores de sintaxis Mermaid: {self.error_reasons['syntax_error']}")
                print("   Revise que cada línea siga 'ID[Texto]' o 'ID{Texto}' y que las flechas sean 'A --> B'.")
            if self.error_reasons['connection_error'] > 0:
                print(f"🔸 Errores de conexión: {self.error_reasons['connection_error']}")
            if self.error_reasons['other'] > 0:
                print(f"🔸 Otros: {self.error_reasons['other']}")
        
        return self.stats

def main():
    """Función principal con argumentos de línea de comandos."""
    parser = argparse.ArgumentParser(
        description="Generador avanzado de diagramas Mermaid a imágenes PNG",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Ejemplos de uso:
  python advanced_diagram_generator.py
  python advanced_diagram_generator.py --docs-dir docs --format svg
  python advanced_diagram_generator.py --pattern "flujo" --format png
        """
    )
    
    parser.add_argument(
        '--docs-dir', 
        default='docs',
        help='Directorio de documentación (default: docs)'
    )
    
    parser.add_argument(
        '--format',
        choices=['png', 'svg', 'pdf'],
        default='png',
        help='Formato de salida (default: png)'
    )

    parser.add_argument(
        '--skip-existing',
        action='store_true',
        help='Omitir la generación si la imagen ya existe'
    )

    parser.add_argument(
        '--force',
        action='store_true',
        help='Forzar regeneración aunque la imagen ya exista'
    )
    
    parser.add_argument(
        '--pattern',
        help='Filtro de archivos por patrón'
    )

    parser.add_argument(
        '--clean',
        action='store_true',
        help='Borrar imágenes generadas antes de generar'
    )

    parser.add_argument(
        '--clean-only',
        action='store_true',
        help='Sólo borrar imágenes generadas y salir'
    )
    
    parser.add_argument(
        '--kroki-url',
        default='https://kroki.io',
        help='URL del servidor Kroki (default: https://kroki.io)'
    )
    
    args = parser.parse_args()
    
    # Crear generador
    # Variables de entorno tienen prioridad si están definidas
    env_skip = os.environ.get('SKIP_EXISTING')
    env_force = os.environ.get('FORCE')
    env_clean = os.environ.get('CLEAN')
    env_clean_only = os.environ.get('CLEAN_ONLY')

    skip_existing = (env_skip.lower() in ['1', 'true', 'yes', 'y']) if env_skip else args.skip_existing or True
    force = (env_force.lower() in ['1', 'true', 'yes', 'y']) if env_force else args.force or False
    clean = (env_clean.lower() in ['1', 'true', 'yes', 'y']) if env_clean else args.clean or False
    clean_only = (env_clean_only.lower() in ['1', 'true', 'yes', 'y']) if env_clean_only else args.clean_only or False

    generator = AdvancedDiagramGenerator(
        base_url=args.kroki_url,
        output_format=args.format,
        skip_existing=skip_existing,
        force=force,
        clean=clean,
        clean_only=clean_only
    )
    
    # Generar diagramas
    stats = generator.generate_all_diagrams(
        docs_dir=args.docs_dir,
        pattern=args.pattern
    )
    
    return stats

if __name__ == "__main__":
    main()