"use client";

import { Button } from "@/components/ui/button";
import type { BudgetRow } from "./page";

interface BudgetActionsCellProps {
  row: BudgetRow;
  budgets: any[];
  onEdit: (budget: any) => void;
  onDelete: (budget: any) => void;
}

export function BudgetActionsCell({ row, budgets, onEdit, onDelete }: BudgetActionsCellProps) {
  const budget = budgets.find(b => b.id === row.id);
  return (
    <div className="flex justify-end gap-2">
      {budget && (
        <>
          <Button variant="outline" size="sm" onClick={() => onEdit(budget)}>Edit</Button>
          <Button variant="destructive" size="sm" onClick={() => onDelete(budget)}>Delete</Button>
        </>
      )}
    </div>
  );
}