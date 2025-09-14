import React from 'react';
import Button from './primitives/Button';

type Props = {
    page: number;
    totalPages: number;
    rangeStart: number; // 1-based index of first item on page
    rangeEnd: number; // 1-based index of last item shown
    totalItems: number;
    onPrev: () => void;
    onNext: () => void;
};

export const Pager: React.FC<Props> = ({
    page,
    totalPages,
    rangeStart,
    rangeEnd,
    totalItems,
    onPrev,
    onNext,
}) => {
    return (
        <div className="pager" role="navigation" aria-label="Pagination">
            <Button
                variant="ghost"
                disabled={page <= 1}
                onClick={onPrev}
                aria-label="Previous page"
            >
                Prev
            </Button>
            <span className="pager-info">
                {totalItems === 0 ? '0' : `${rangeStart}`}–{rangeEnd} of{' '}
                {totalItems}
            </span>
            <Button
                variant="ghost"
                disabled={page >= totalPages}
                onClick={onNext}
                aria-label="Next page"
            >
                Next
            </Button>
        </div>
    );
};

export default Pager;
