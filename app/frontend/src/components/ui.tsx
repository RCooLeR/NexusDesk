import type {ButtonHTMLAttributes, ReactNode} from 'react';

type ButtonVariant = 'primary' | 'secondary' | 'subtle';

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
    variant?: ButtonVariant;
};

type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
    label: string;
};

type StatusBadgeProps = {
    children: ReactNode;
    tone?: 'neutral' | 'warning' | 'success';
};

export function Button({children, className = '', type = 'button', variant = 'secondary', ...props}: ButtonProps) {
    return (
        <button className={mergeClassNames('ui-button', `ui-button-${variant}`, className)} type={type} {...props}>
            {children}
        </button>
    );
}

export function IconButton({children, className = '', label, title, type = 'button', ...props}: IconButtonProps) {
    return (
        <button
            aria-label={label}
            className={mergeClassNames('ui-button', 'ui-icon-button', className)}
            title={title ?? label}
            type={type}
            {...props}
        >
            {children}
        </button>
    );
}

export function StatusBadge({children, tone = 'neutral'}: StatusBadgeProps) {
    return <span className={mergeClassNames('status-badge', `status-badge-${tone}`)}>{children}</span>;
}

function mergeClassNames(...classNames: string[]) {
    return classNames.filter(Boolean).join(' ');
}
