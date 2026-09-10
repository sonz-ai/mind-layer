#!/usr/bin/env python3
"""Generate standalone SVG diagrams from actual recorded scenario/benchmark data."""
import html
import json
from pathlib import Path


def build(root):
    assets = root / "docs/assets"
    assets.mkdir(exist_ok=True)
    def svg(title, content, height=340):
        return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 900 {height}" role="img" aria-label="{html.escape(title)}"><title>{html.escape(title)}</title><rect width="900" height="{height}" rx="16" fill="#f5f7f0"/><g font-family="system-ui,sans-serif" fill="#20382d">{content}</g></svg>\n'
    def text(x,y,value,size=16,color="#20382d"):
        return f'<text x="{x}" y="{y}" font-size="{size}" fill="{color}">{html.escape(str(value))}</text>'
    def box(x,y,w,label,sub):
        return f'<rect x="{x}" y="{y}" width="{w}" height="76" rx="10" fill="white" stroke="#9eb4a0"/>'+text(x+16,y+30,label,18)+text(x+16,y+55,sub,13)
    architecture = text(30,40,"Mind Layer: local ownership, optional model calls",24)
    architecture += box(30,85,245,"Your application","REST client or character example")+box(335,85,245,"Mind Layer","Memory + retrieval + personality")+box(640,85,225,"Local database","Embedded Bolt; no server")
    for x1,x2 in [(275,335),(580,640)]:
        architecture += f'<path d="M{x1} 123 H{x2-10}" stroke="#337357" stroke-width="3"/><path d="M{x2-15} 117 L{x2-5} 123 L{x2-15} 129" fill="none" stroke="#337357" stroke-width="3"/>'
    architecture += box(335,220,320,"Your chosen model provider","Optional embedding, extraction and replies")
    architecture += '<path d="M455 161 V212" stroke="#b36920" stroke-width="3" stroke-dasharray="5 4"/>'+text(480,194,"Direct API calls",14)
    architecture += text(30,325,"No Sonzai account, billing service, telemetry or license-server dependency.",15)
    (assets / "architecture.svg").write_text(svg("Mind Layer architecture",architecture))
    result = json.loads((root / "benchmarks/results/character-scenario.json").read_text())["character"]
    points = [{"trust":20,"confidence":30,"curiosity":45}]+[e["after"] for e in result["events"]]
    labels=["Start"]+[e["action"]["id"] for e in result["events"]]
    graph=text(30,38,"Moss evolves through five remembered interactions",24)
    for value in [0,25,50,75,100]:
        y=270-value*1.8
        graph+=f'<line x1="75" y1="{y}" x2="830" y2="{y}" stroke="#d6dfd1"/>'+text(35,y+5,value,13)
    for key,color in [("trust","#337357"),("confidence","#b36920"),("curiosity","#5869b5")]:
        coords=[(75+i*151,270-p[key]*1.8) for i,p in enumerate(points)]
        graph+=f'<polyline points="{" ".join(f"{x},{y}" for x,y in coords)}" fill="none" stroke="{color}" stroke-width="3"/>'
        for x,y in coords:graph+=f'<circle cx="{x}" cy="{y}" r="4" fill="{color}"/>'
    for i,label in enumerate(labels):graph+=text(55+i*151,300,label,12)
    for i,(label,color) in enumerate([("Trust","#337357"),("Confidence","#b36920"),("Curiosity","#5869b5")]):graph+=text(75+i*180,330,label,15,color)
    graph+=text(30,365,"Deterministic fictional game traits (0–100), not inferred psychological measurements.",14)
    (assets / "character-evolution.svg").write_text(svg("Moss trait evolution",graph,390))
    lex=json.loads((root/"benchmarks/results/lexical.json").read_text());hybrid=json.loads((root/"benchmarks/results/hybrid.json").read_text())
    graph=text(30,40,"Retrieval diagnostic: mean recall@3",24)+text(30,68,"12 authored synthetic queries; not LoCoMo, LongMemEval or answer accuracy.",15)
    for i,(label,value,color) in enumerate([("Recency baseline",lex["recency_baseline_recall_at_k"],"#a5afa2"),("BM25 keyword",lex["mean_recall_at_k"],"#739a72"),("Hybrid + Gemini",hybrid["mean_recall_at_k"],"#337357")]):
        y=110+i*65;graph+=text(30,y+24,label,17)+f'<rect x="245" y="{y}" width="{value*540}" height="34" rx="5" fill="{color}"/>'+text(795,y+24,f"{value:.0%}",18)
    graph+=text(30,320,"See benchmarks/results/ for raw measurements, timestamps, models and dataset hash.",14)
    (assets / "retrieval-benchmark.svg").write_text(svg("Measured synthetic retrieval recall",graph))


if __name__ == "__main__":
    build(Path(__file__).resolve().parents[1])
