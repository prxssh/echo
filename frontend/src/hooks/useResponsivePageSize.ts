import { useEffect, useState } from 'react';

export function useResponsivePageSize(
    small: number,
    large: number,
    breakpoint = 1200
) {
    const [pageSize, setPageSize] = useState<number>(small);

    useEffect(() => {
        const compute = () =>
            setPageSize(window.innerWidth >= breakpoint ? large : small);
        compute();
        window.addEventListener('resize', compute);
        return () => window.removeEventListener('resize', compute);
    }, [small, large, breakpoint]);

    return pageSize;
}

export default useResponsivePageSize;
