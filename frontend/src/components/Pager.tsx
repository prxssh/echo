import React from 'react';

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
            <button
                className="btn-ghost"
                disabled={page <= 1}
                onClick={onPrev}
                aria-label="Previous page"
            >
                Prev
            </button>
            <span className="pager-info">
                {totalItems === 0 ? '0' : `${rangeStart}`}–{rangeEnd} of{' '}
                {totalItems}
            </span>
            <button
                className="btn-ghost"
                disabled={page >= totalPages}
                onClick={onNext}
                aria-label="Next page"
            >
                Next
            </button>
        </div>
    );
};

export default Pager;
