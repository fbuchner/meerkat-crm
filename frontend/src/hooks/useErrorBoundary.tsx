import { Component, ReactNode } from 'react';
import ErrorBoundary from '../components/ErrorBoundary';
import { ErrorFallback, SectionErrorFallback } from '../components/ErrorFallback';

/**
 * Higher-order component to wrap a component with an error boundary
 * 
 * @example
 * ```tsx
 * const SafeContactsList = withErrorBoundary(ContactsList, {
 *   name: 'ContactsList',
 *   fallbackSize: 'medium'
 * });
 * ```
 */
export function withErrorBoundary<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  options: {
    name?: string;
    fallbackSize?: 'small' | 'medium' | 'large';
    showDetails?: boolean;
    onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  } = {}
) {
  const { name, fallbackSize = 'medium', showDetails, onError } = options;
  
  return class WithErrorBoundary extends Component<P> {
    render() {
      return (
        <ErrorBoundary
          name={name || WrappedComponent.displayName || WrappedComponent.name}
          showDetails={showDetails ?? process.env.NODE_ENV === 'development'}
          onError={onError}
          fallback={
            <ErrorFallback
              title={`Error in ${name || 'component'}`}
              size={fallbackSize}
              showDetails={showDetails ?? process.env.NODE_ENV === 'development'}
            />
          }
        >
          <WrappedComponent {...this.props} />
        </ErrorBoundary>
      );
    }
  };
}

/**
 * Higher-order component for minimal section-level error boundaries
 * 
 * @example
 * ```tsx
 * const SafeSection = withSectionErrorBoundary(MySection);
 * ```
 */
export function withSectionErrorBoundary<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  options: {
    name?: string;
    onError?: (error: Error, errorInfo: React.ErrorInfo) => void;
  } = {}
) {
  const { name, onError } = options;
  
  return class WithSectionErrorBoundary extends Component<P> {
    render() {
      return (
        <ErrorBoundary
          name={name || WrappedComponent.displayName || WrappedComponent.name}
          onError={onError}
          fallback={
            <SectionErrorFallback
              title={`Error loading ${name || 'section'}`}
            />
          }
        >
          <WrappedComponent {...this.props} />
        </ErrorBoundary>
      );
    }
  };
}

/**
 * Props for SafeComponent wrapper
 */
interface SafeComponentProps {
  children: ReactNode;
  name?: string;
  fallbackSize?: 'small' | 'medium' | 'large';
  showDetails?: boolean;
}

/**
 * Simple wrapper component for adding error boundaries inline
 * 
 * @example
 * ```tsx
 * <SafeComponent name="ContactCard">
 *   <ContactCard contact={contact} />
 * </SafeComponent>
 * ```
 */
export const SafeComponent: React.FC<SafeComponentProps> = ({
  children,
  name = 'Component',
  fallbackSize = 'small',
  showDetails,
}) => {
  return (
    <ErrorBoundary
      name={name}
      showDetails={showDetails ?? process.env.NODE_ENV === 'development'}
      fallback={
        <ErrorFallback
          title={`Error in ${name}`}
          size={fallbackSize}
          showDetails={showDetails ?? process.env.NODE_ENV === 'development'}
        />
      }
    >
      {children}
    </ErrorBoundary>
  );
};

/**
 * Simple wrapper for section-level errors
 * 
 * @example
 * ```tsx
 * <SafeSection name="Statistics Panel">
 *   <StatisticsPanel />
 * </SafeSection>
 * ```
 */
export const SafeSection: React.FC<SafeComponentProps> = ({
  children,
  name = 'Section',
}) => {
  return (
    <ErrorBoundary
      name={name}
      fallback={<SectionErrorFallback title={`Error loading ${name}`} />}
    >
      {children}
    </ErrorBoundary>
  );
};

export default {
  withErrorBoundary,
  withSectionErrorBoundary,
  SafeComponent,
  SafeSection,
};
