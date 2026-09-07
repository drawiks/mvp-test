import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import { EditorView, keymap } from "@codemirror/view";
import { Compartment, EditorState, Transaction } from "@codemirror/state";
import { formulaLanguage, formulaAutocompletion, formulaTheme, createFormulaLinter } from "@/lib/formulaLang";
import { cn } from "@/lib/utils";

type Props = {
  expression: string;
  onChange: (expr: string) => void;
  readOnly?: boolean;
  className?: string;
  height?: string;
};

export type FormulaEditorHandle = {
  insert: (text: string) => void;
};

export default forwardRef<FormulaEditorHandle, Props>(function FormulaEditor(
  { expression, onChange, readOnly = false, className, height = "200px" },
  ref
) {
  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const onChangeRef = useRef(onChange);
  onChangeRef.current = onChange;
  const readOnlyRef = useRef(readOnly);
  readOnlyRef.current = readOnly;
  const editableCompartment = useRef(new Compartment()).current;

  useEffect(() => {
    if (!editorRef.current) return;
    const view = new EditorView({
      parent: editorRef.current,
      state: EditorState.create({
        doc: expression,
        extensions: [
          formulaLanguage,
          formulaAutocompletion(),
          createFormulaLinter(),
          formulaTheme,
          EditorView.lineWrapping,
          editableCompartment.of(EditorView.editable.of(!readOnlyRef.current)),
          keymap.of([
            {
              key: "Tab",
              run: (v) => {
                const sel = v.state.selection.main;
                if (sel.empty && !readOnlyRef.current) {
                  v.dispatch({ changes: { from: sel.from, to: sel.to, insert: "  " } });
                  return true;
                }
                return false;
              },
            },
          ]),
          EditorView.updateListener.of((update) => {
            if (update.docChanged && update.transactions.some((t) => t.annotation(Transaction.userEvent))) {
              onChangeRef.current(update.state.doc.toString());
            }
          }),
        ],
      }),
    });
    viewRef.current = view;
    return () => {
      view.destroy();
      viewRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const v = viewRef.current;
    if (v && expression !== v.state.doc.toString()) {
      v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: expression } });
    }
  }, [expression]);

  useEffect(() => {
    const v = viewRef.current;
    if (v) {
      v.dispatch({ effects: editableCompartment.reconfigure(EditorView.editable.of(!readOnlyRef.current)) });
    }
  }, [readOnly, editableCompartment]);

  useImperativeHandle(
    ref,
    () => ({
      insert(text: string) {
        const v = viewRef.current;
        if (!v) return;
        v.dispatch(v.state.replaceSelection(text));
        v.focus();
      },
    }),
    []
  );

  return (
    <div
      ref={editorRef}
      className={cn("border border-border rounded-md bg-background font-mono text-[12px] overflow-hidden", readOnly && "opacity-80", className)}
      style={{ height, minHeight: height }}
    />
  );
});