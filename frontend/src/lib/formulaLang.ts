import { LanguageSupport, StreamLanguage, HighlightStyle, syntaxHighlighting, type StreamParser } from "@codemirror/language";
import { EditorView } from "@codemirror/view";
import { tags } from "@lezer/highlight";
import { autocompletion, type CompletionContext } from "@codemirror/autocomplete";
import { linter, type Diagnostic } from "@codemirror/lint";
import { WEIGHT_TOKENS } from "@/lib/weights";
import * as Bindings from "../../wailsjs/go/main/App";

const statTokens = WEIGHT_TOKENS.map((m) => m.key);
const funcTokens = ["max", "min", "abs", "round"];
const keywordTokens = ["if", "else", "elif", "true", "false"];
const operators = ["+", "-", "*", "/", "**", "==", "!=", "<", ">", "<=", ">=", "&&", "||", "!", "?", ":", "{", "}", "(", ")"];

const formulaParser: StreamParser<object> = {
  name: "formula",
  token(stream) {
    if (stream.eatSpace()) return null;
    if (stream.match(/#[^\n]*/) || stream.match(/\/\/[^\n]*/)) return "comment";
    if (stream.match(/\d+(\.\d+)?/)) return "number";
    const start = stream.pos;
    if (stream.match(/[a-zA-Z_][a-zA-Z0-9_]*/)) {
      const w = stream.string.slice(start, stream.pos);
      if (w === "if" || w === "else" || w === "elif") return "keyword";
      if (w === "true" || w === "false") return "boolean";
      if (funcTokens.includes(w)) return "propertyName";
      return "variableName";
    }
    if (stream.match(/\*\*|==|!=|<=|>=|&&|\|\|/) || stream.match(/[+\-*/<>!]/)) return "operator";
    if (stream.match(/[{}()?:,]/)) return "punctuation";
    stream.next();
    return "invalid";
  },
  startState() {
    return {};
  },
};

export const formulaLanguage = new LanguageSupport(StreamLanguage.define(formulaParser));

export const formulaHighlightStyle = HighlightStyle.define([
  { tag: tags.comment, color: "#6e7681", fontStyle: "italic" },
  { tag: tags.number, color: "#e3b341" },
  { tag: tags.keyword, color: "#f5c542" },
  { tag: tags.bool, color: "#f5c542" },
  { tag: tags.propertyName, color: "#58a6ff" },
  { tag: tags.variableName, color: "#7ee787" },
  { tag: tags.operator, color: "#e6edf3" },
  { tag: tags.punctuation, color: "#8b949e" },
]);

export const formulaDarkTheme = EditorView.theme(
  {
    "&": { color: "#e6edf3", backgroundColor: "transparent", height: "100%" },
    ".cm-scroller": { overflow: "auto", lineHeight: "1.6" },
    ".cm-content": { caretColor: "#f5c542", fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace", padding: "4px 0" },
    ".cm-cursor, .cm-dropCursor": { borderLeftColor: "#f5c542" },
    "&.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-selectionBackground, ::selection": {
      backgroundColor: "#2a4365",
    },
    ".cm-activeLine": { backgroundColor: "rgba(148,163,184,0.06)" },
    ".cm-gutters": {
      backgroundColor: "transparent",
      color: "#6e7681",
      border: "none",
    },
    ".cm-tooltip": {
      backgroundColor: "#161b22",
      border: "1px solid #232a35",
      color: "#e6edf3",
    },
    ".cm-tooltip-autocomplete > ul > li[aria-selected]": {
      backgroundColor: "#1f6feb",
      color: "#fff",
    },
    ".cm-tooltip.cm-tooltip-autocomplete > ul > li": { color: "#e6edf3" },
    ".cm-diagnostic": { borderLeft: "2px solid #f85149" },
    ".cm-diagnostic-error": { backgroundColor: "rgba(248,81,73,0.12)" },
  },
  { dark: true }
);

export const formulaTheme = [formulaDarkTheme, syntaxHighlighting(formulaHighlightStyle)];

export function formulaAutocomplete(context: CompletionContext) {
  const word = context.matchBefore(/\w+/);
  if (!word || !word.text) return null;
  const options = [...statTokens, ...funcTokens, ...keywordTokens, ...operators]
    .filter((t) => t.startsWith(word.text))
    .map((label) => ({ label, type: "keyword" as const }));
  return { from: word.from, options };
}

export function formulaAutocompletion() {
  return autocompletion({ override: [formulaAutocomplete] });
}

export async function lintFormula(expr: string): Promise<Diagnostic[]> {
  if (!expr.trim()) return [];
  try {
    await Bindings.ValidateExpression(expr);
    return [];
  } catch (e) {
    const msg = String(e);
    return [{ from: 0, to: expr.length, severity: "error", message: msg }];
  }
}

export function createFormulaLinter() {
  return linter((view) => lintFormula(view.state.doc.toString()), { delay: 300 });
}