import { create } from "zustand";
import { StateCreator } from "zustand";

export type ToastVariant = "default" | "destructive";

export interface Toast {
	id: string;
	title?: string;
	description?: string;
	variant?: ToastVariant;
	duration?: number;
}

interface ToastState {
	toasts: Toast[];
	addToast: (toast: Omit<Toast, "id">) => void;
	removeToast: (id: string) => void;
}

const toastCreator: StateCreator<ToastState> = (set) => ({
	toasts: [],
	addToast: (toast) => {
		const id = Math.random().toString(36).substring(2, 9);
		const newToast = { ...toast, id, duration: toast.duration || 5000 };
		set((state) => ({ toasts: [...state.toasts, newToast] }));

		if (newToast.duration) {
			setTimeout(() => {
				set((state) => ({
					toasts: state.toasts.filter((t) => t.id !== id),
				}));
			}, newToast.duration);
		}
	},
	removeToast: (id) => {
		set((state) => ({ toasts: state.toasts.filter((t) => t.id !== id) }));
	},
});

export const useToastStore = create<ToastState>(toastCreator);
