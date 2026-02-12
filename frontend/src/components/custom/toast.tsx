import React, { useEffect } from "react";
import { X } from "lucide-react";
import { useToastStore } from "@/lib/toastStore";
import { cn } from "@/lib/utils";

export const Toast = () => {
	const { toasts, removeToast } = useToastStore();

	if (toasts.length === 0) return null;

	return (
		<div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
			{toasts.map((toast) => (
				<div
					key={toast.id}
					className={cn(
						"flex items-start gap-3 rounded-lg border p-4 shadow-lg transition-all",
						"bg-white dark:bg-slate-900",
						toast.variant === "destructive"
							? "border-red-200 dark:border-red-900"
							: "border-slate-200 dark:border-slate-800",
						"animate-in slide-in-from-right-5 fade-in-5 duration-300",
					)}
				>
					<div className="flex-1">
						{toast.title && (
							<h4 className="text-sm font-semibold">{toast.title}</h4>
						)}
						{toast.description && (
							<p className="text-sm text-slate-600 dark:text-slate-400">
								{toast.description}
							</p>
						)}
					</div>
					<button
						onClick={() => removeToast(toast.id)}
						className="text-slate-400 hover:text-slate-600 transition-colors"
					>
						<X className="h-4 w-4" />
					</button>
				</div>
			))}
		</div>
	);
};
