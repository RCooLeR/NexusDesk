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

type CardProps = {
    children: ReactNode;
    className?: string;
};

type StateTone = 'neutral' | 'warning' | 'danger';

type StateProps = {
    detail?: string;
    iconSrc?: string;
    title: string;
    tone?: StateTone;
};

type InlineAlertProps = {
    children: ReactNode;
    tone?: StateTone;
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

export function Card({children, className = ''}: CardProps) {
    return <section className={mergeClassNames('ui-card', className)}>{children}</section>;
}

export function EmptyState({detail, iconSrc, title, tone = 'neutral'}: StateProps) {
    return (
        <div className={mergeClassNames('state-panel', `state-panel-${tone}`)}>
            {iconSrc && <img src={iconSrc} alt="" />}
            <strong>{title}</strong>
            {detail && <p>{detail}</p>}
        </div>
    );
}

export function LoadingState({detail, iconSrc, title}: StateProps) {
    return (
        <div className="state-panel state-panel-loading">
            {iconSrc && <img src={iconSrc} alt="" />}
            <strong>{title}</strong>
            {detail && <p>{detail}</p>}
            <span className="state-loader" aria-hidden="true" />
        </div>
    );
}

export function InlineAlert({children, tone = 'neutral'}: InlineAlertProps) {
    return <div className={mergeClassNames('inline-alert', `inline-alert-${tone}`)}>{children}</div>;
}

function mergeClassNames(...classNames: string[]) {
    return classNames.filter(Boolean).join(' ');
}
